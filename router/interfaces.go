package router

import (
	"context"

	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
)

type MQTTRouter interface {
	ProccessMessageFromDevice(devMsg mqtt.DeviceMessage)
	ProccessDeviceDescriptionMessage(devDscMsg mqtt.DeviceDescriptionMessage)
	SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage))
	SubscribeOnGetMessage(callback func(devCmd types.DeviceGetMessage))
	SubscribeOnExploreMessage(callback func(devCmd types.DeviceExploreMessage))
	SubscribeOnSetDeviceConfigMessage(callback func(devCmd types.DeviceConfigSetMessage))
}

type ZigbeeRouter interface {
	SubscribeOnDeviceMessage(callback func(devMsg mqtt.DeviceMessage))
	SubscribeOnDeviceDescription(callback func(devMsg mqtt.DeviceDescriptionMessage))
	ProccessMessageToDevice(ctx context.Context, devCmd types.DeviceCommandMessage)
	ProccessGetMessageToDevice(ctx context.Context, devCmd types.DeviceGetMessage)
	ProccessSetDeviceConfigMessage(ctx context.Context, devCmd types.DeviceConfigSetMessage)
	ProccessGetDeviceDescriptionMessage(ctx context.Context, devCmd types.DeviceExploreMessage)
	StartAsync(ctx context.Context)
}
