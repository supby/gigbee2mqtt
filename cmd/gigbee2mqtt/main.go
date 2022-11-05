package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/shimmeringbee/zigbee"

	"github.com/supby/gigbee2mqtt/internal/configuration"
	"github.com/supby/gigbee2mqtt/internal/db"
	"github.com/supby/gigbee2mqtt/internal/logger"
	"github.com/supby/gigbee2mqtt/internal/mqtt"
	"github.com/supby/gigbee2mqtt/internal/router"
	"github.com/supby/gigbee2mqtt/internal/types"
	"github.com/supby/gigbee2mqtt/internal/zcldef"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logger.GetLogger("[main]", logger.LogLevelError)

	var configFile = flag.String("c", "./configuration.yaml", "path to config file name")
	flag.Parse()

	configService, err := configuration.Init(*configFile)
	if err != nil {
		logger.Error("Configuration initialization error: %v\n", err)
		os.Exit(1)
	}

	db1, err := db.NewDeviceDB("./data", db.DeviceDBOptions{
		FlushPeriodInSeconds: 60,
	})
	if err != nil {
		logger.Error("db initialization error: %v\n", err)
		os.Exit(1)
	}
	defer db1.Close(ctx)

	zclDefService := zcldef.New("./zcldef/zcldef.json")

	cfg := configService.GetConfiguration()

	mqttClient, mqttDisconnect := mqtt.NewClient(&cfg)
	defer mqttDisconnect()

	mqttRouter := router.NewMQTTRouter(configService, mqttClient, db1)
	zRouter := router.NewZigbeeRouter(zclDefService, db1, &cfg)

	setupSubscriptions(mqttRouter, zRouter, ctx)

	zRouter.StartAsync(ctx)
	defer zRouter.Stop()

	waitForInterruptSignal()

	logger.Info("exiting app...")
}

func setupSubscriptions(mqttRouter router.MQTTRouter, zRouter router.ZigbeeRouter, ctx context.Context) {
	mqttRouter.SubscribeOnCommandMessage(func(devCmd types.DeviceCommandMessage) {
		zRouter.ProccessCommandMessageToDevice(ctx, devCmd)
	})
	mqttRouter.SubscribeOnSetMessage(func(devCmd types.DeviceSetMessage) {
		zRouter.ProccessSetMessageToDevice(ctx, devCmd)
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
	zRouter.SubscribeOnAdapterInitialized(func(e zigbee.Node) {
		mqttRouter.PublishGatewayMessage(e, "")
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
}

func waitForInterruptSignal() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	defer func() {
		signal.Stop(sigchan)
	}()
	<-sigchan
}
