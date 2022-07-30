package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/level"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/logger"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/router"
	"github.com/supby/gigbee2mqtt/types"
	"github.com/supby/gigbee2mqtt/zcldef"

	"go.bug.st/serial.v1"
)

func main() {
	logger := logger.GetLogger("[main]")

	var configFile = flag.String("c", "./configuration.yaml", "path to config file name")
	flag.Parse()

	pctx := context.Background()

	configService, err := configuration.Init(*configFile)
	if err != nil {
		logger.Log("Configuration initialization error: %v\n", err)
		os.Exit(1)
	}

	db1 := db.Init(db.DBOption{
		Filename:   "./data/db.json",
		FlushAfter: 10,
	})

	cfg := configService.GetConfiguration()

	z := initZStack(pctx, &cfg, db1)
	defer z.Stop()

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)
	onoff.Register(zclCommandRegistry)
	level.Register(zclCommandRegistry)
	// TODO: register all clusters

	zclDefService := zcldef.New("./zcldef/zcldef.json")

	mqttClient, mqttDisconnect := mqtt.NewClient(&cfg)
	defer mqttDisconnect()

	mqttRouter := router.NewMQTTRouter(configService, mqttClient, db1)
	zRouter := router.NewZigbeeRouter(z, zclCommandRegistry, zclDefService, db1, &cfg)

	ctx, cancel := context.WithCancel(pctx)

	mqttRouter.SubscribeOnSetMessage(func(devCmd types.DeviceCommandMessage) {
		zRouter.ProccessMessageToDevice(ctx, devCmd)
	})
	mqttRouter.SubscribeOnGetMessage(func(devCmd types.DeviceGetMessage) {
		zRouter.ProccessGetMessageToDevice(ctx, devCmd)
	})
	mqttRouter.SubscribeOnExploreMessage(func(devCmd types.DeviceExploreMessage) {
		zRouter.ProccessGetDeviceDescriptionMessage(ctx, devCmd)
	})
	mqttRouter.SubscribeOnSetDeviceConfigMessage(func(devCmd types.DeviceConfigSetMessage) {
		zRouter.ProccessSetDeviceConfigMessage(ctx, devCmd)
	})
	zRouter.SubscribeOnDeviceMessage(func(devMsg mqtt.DeviceMessage) {
		mqttRouter.PublishDeviceMessage(devMsg.IEEEAddress, devMsg, "")
	})
	zRouter.SubscribeOnDeviceDescription(func(devDscMsg mqtt.DeviceDescriptionMessage) {
		mqttRouter.PublishDeviceMessage(devDscMsg.IEEEAddress, devDscMsg, "description")
	})

	zRouter.SubscribeOnDeviceJoin(func(e zigbee.NodeJoinEvent) {
		mqttRouter.PublishDeviceMessage(uint64(e.IEEEAddress), e, "join")
	})
	zRouter.SubscribeOnDeviceLeave(func(e zigbee.NodeLeaveEvent) {
		mqttRouter.PublishDeviceMessage(uint64(e.IEEEAddress), e, "leave")
	})
	zRouter.SubscribeOnDeviceUpdate(func(e zigbee.NodeUpdateEvent) {
		mqttRouter.PublishDeviceMessage(uint64(e.IEEEAddress), e, "update")
	})

	zRouter.StartAsync(ctx)

	waitForSignal(cancel)

	logger.Log("exiting app...")
}

func waitForSignal(cancel context.CancelFunc) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	defer func() {
		cancel()
		signal.Stop(sigchan)
	}()
	<-sigchan
}

func initZStack(pctx context.Context, cfg *configuration.Configuration, db1 db.DevicesRepo) *zstack.ZStack {
	logger := logger.GetLogger("[init zstack]")

	mode := &serial.Mode{
		BaudRate: int(cfg.SerialConfiguration.BaudRate),
	}

	port, err := serial.Open(cfg.SerialConfiguration.PortName, mode)
	if err != nil {
		log.Fatal(err)
	}
	port.SetRTS(true)

	/* Construct node table, cache of network nodes. */
	dbNodes := db1.GetNodes()
	t := zstack.NewNodeTable()
	znodes := make([]zigbee.Node, len(dbNodes))
	for i, dbNode := range dbNodes {
		znodes[i] = zigbee.Node{
			IEEEAddress:    zigbee.IEEEAddress(dbNode.IEEEAddress),
			NetworkAddress: zigbee.NetworkAddress(dbNode.NetworkAddress),
			LogicalType:    zigbee.LogicalType(dbNode.LogicalType),
			LQI:            dbNode.LQI,
			Depth:          dbNode.Depth,
			LastDiscovered: dbNode.LastDiscovered,
			LastReceived:   dbNode.LastReceived,
		}
	}
	t.Load(znodes)

	/* Create a new ZStack struct. */
	z := zstack.New(port, t)

	netCfg := zigbee.NetworkConfiguration{
		PANID:         zigbee.PANID(cfg.ZNetworkConfiguration.PANID),
		ExtendedPANID: zigbee.ExtendedPANID(cfg.ZNetworkConfiguration.ExtendedPANID),
		NetworkKey:    cfg.ZNetworkConfiguration.NetworkKey,
		Channel:       cfg.ZNetworkConfiguration.Channel,
	}

	/* Obtain context for timeout of initialisation. */
	initCtx, cancel := context.WithTimeout(pctx, 2*time.Minute)
	defer cancel()

	/* Initialise ZStack and CC253X */
	err = z.Initialise(initCtx, netCfg)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.PermitJoin {
		err = z.PermitJoin(initCtx, true)
		if err != nil {
			logger.Log("error permit join: %v\n", err)
		}
	} else {
		err = z.DenyJoin(initCtx)
		if err != nil {
			logger.Log("error deny join: %v\n", err)
		}
	}

	if err := z.RegisterAdapterEndpoint(initCtx, zigbee.Endpoint(0x01), zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		log.Fatal(err)
	}

	return z
}
