package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/shimmeringbee/zigbee"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/logger"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/router"
	"github.com/supby/gigbee2mqtt/types"
	"github.com/supby/gigbee2mqtt/zcldef"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logger.GetLogger("[main]")

	var configFile = flag.String("c", "./configuration.yaml", "path to config file name")
	flag.Parse()

	configService, err := configuration.Init(*configFile)
	if err != nil {
		logger.Log("Configuration initialization error: %v\n", err)
		os.Exit(1)
	}

	db1, err := db.NewDeviceDB("./data")
	if err != nil {
		logger.Log("db initialization error: %v\n", err)
		os.Exit(1)
	}

	zclDefService := zcldef.New("./zcldef/zcldef.json")

	cfg := configService.GetConfiguration()

	mqttClient, mqttDisconnect := mqtt.NewClient(&cfg)
	defer mqttDisconnect()

	mqttRouter := router.NewMQTTRouter(configService, mqttClient, db1)
	zRouter := router.NewZigbeeRouter(zclDefService, db1, &cfg)

	setupSubscriptions(mqttRouter, zRouter, ctx)

	zRouter.StartAsync(ctx)
	defer zRouter.Stop()

	waitForSignal(cancel)

	logger.Log("exiting app...")
}

func setupSubscriptions(mqttRouter router.MQTTRouter, zRouter router.ZigbeeRouter, ctx context.Context) {
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
