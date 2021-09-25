package main

import (
	"context"
	"log"
	"time"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/level"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/service"
	"github.com/supby/gigbee2mqtt/types"
	"github.com/supby/gigbee2mqtt/zcldef"

	"go.bug.st/serial.v1"
)

// Example of message:
// MQTT publish: topic 'zigbee2mqtt/0x00124b000724ae04',
// payload '{"cluster":"ssIasZone","data":{"batteryPercentageRemaining":200,"batteryVoltage":30,"zoneStatus":1},"device":{"friendlyName":"0x00124b000724ae04","ieeeAddr":"0x00124b000724ae04","model":"unknown","networkAddress":31211,"type":"Unknown"},"endpoint":{"ID":1,"_binds":[],"_configuredReportings":[],"clusters":{"genPowerCfg":{"attributes":{"batteryPercentageRemaining":200,"batteryVoltage":30}},"ssIasZone":{"attributes":{"zoneStatus":1}}},"deviceIeeeAddress":"0x00124b000724ae04","deviceNetworkAddress":31211,"inputClusters":[],"meta":{},"outputClusters":[]},"groupID":0,"linkquality":52,"meta":{"frameControl":{"direction":1,"disableDefaultResponse":true,"frameType":0,"manufacturerSpecific":false,"reservedBits":0},"manufacturerCode":null,"zclTransactionSequenceNumber":219},"type":"attributeReport"}'

func main() {
	pctx := context.Background()

	cfg := configuration.Init("./configuration_livolo.yaml")

	db1 := db.Init("./db.json")

	z := initZStack(pctx, cfg, db1)

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)
	onoff.Register(zclCommandRegistry)
	level.Register(zclCommandRegistry)

	zclDefMap := zcldef.Load("./zcldef/zcldef.json")
	if zclDefMap == nil {
		log.Fatal("Error loading ZCL map")
	}

	mqttClient, mqttDisconnect := mqtt.NewClient(cfg)
	defer mqttDisconnect()

	mqttService := service.CreateMQTTMessageService(cfg, mqttClient)
	zService := service.CreateZigbeeMessageService(z, zclCommandRegistry, zclDefMap, db1, cfg)

	// TODO: move to separate router
	mqttService.SubscribeOnSetMessage(func(devCmd types.DeviceCommandMessage) {
		zService.ProccessMessageToDevice(devCmd)
	})
	zService.SubscribeOnAttributesReport(func(devMsg mqtt.DeviceAttributesReportMessage) {
		mqttService.ProccessMessageFromDevice(devMsg)
	})

	zService.StartAsync(pctx)

	<-pctx.Done()

	log.Println("Exiting app...")
}

func initZStack(pctx context.Context, cfg *configuration.Configuration, db1 *db.DB) *zstack.ZStack {
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
			log.Printf("Error permit join: %v\n", err)
		}
	}

	if err := z.RegisterAdapterEndpoint(initCtx, zigbee.Endpoint(0x01), zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		log.Fatal(err)
	}

	return z
}
