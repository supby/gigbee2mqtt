package router

import (
	"context"

	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
)

type MQTTRouter interface {
	ProccessMessageFromDevice(devMsg mqtt.DeviceMessage)
	SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage))
}

type ZigbeeRouter interface {
	SubscribeOnDeviceMessage(callback func(devMsg mqtt.DeviceMessage))
	ProccessMessageToDevice(ctx context.Context, devCmd types.DeviceCommandMessage)
	StartAsync(ctx context.Context)
}
