package router

import (
	"context"

	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
)

type MQTTRouter interface {
	ProccessMessageFromDevice(devMsg mqtt.DeviceMessage)
	SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage))
	SubscribeOnGetMessage(callback func(devCmd types.DeviceGetMessage))
	SubscribeOnSetDeviceConfigMessage(callback func(devCmd types.DeviceConfigSetMessage))
}

type ZigbeeRouter interface {
	SubscribeOnDeviceMessage(callback func(devMsg mqtt.DeviceMessage))
	ProccessMessageToDevice(ctx context.Context, devCmd types.DeviceCommandMessage)
	ProccessGetMessageToDevice(ctx context.Context, devCmd types.DeviceGetMessage)
	ProccessSetDeviceConfigMessage(ctx context.Context, devCmd types.DeviceConfigSetMessage)
	StartAsync(ctx context.Context)
}