package router

import (
	"context"

	"github.com/shimmeringbee/zigbee"
	"github.com/supby/gigbee2mqtt/internal/mqtt"
	"github.com/supby/gigbee2mqtt/internal/types"
)

type MQTTRouter interface {
	PublishDeviceMessage(ieeeAddress uint64, msg interface{}, subtopic string)
	PublishGatewayMessage(msg interface{}, subtopic string)

	SubscribeOnCommandMessage(callback func(devCmd types.DeviceCommandMessage))
	SubscribeOnGetMessage(callback func(devCmd types.DeviceGetMessage))
	SubscribeOnSetMessage(callback func(devCmd types.DeviceSetMessage))
	SubscribeOnExploreMessage(callback func(devCmd types.DeviceExploreMessage))
	SubscribeOnSetDeviceConfigMessage(callback func(devCmd types.DeviceConfigSetMessage))
}

type ZigbeeRouter interface {
	SubscribeOnAdapterInitialized(callback func(e zigbee.Node))
	SubscribeOnDeviceMessage(callback func(devMsg mqtt.DeviceMessage))
	SubscribeOnDeviceDescription(callback func(devMsg mqtt.DeviceDescriptionMessage))
	SubscribeOnDeviceJoin(cb func(e zigbee.NodeJoinEvent))
	SubscribeOnDeviceLeave(cb func(e zigbee.NodeLeaveEvent))
	SubscribeOnDeviceUpdate(cb func(e zigbee.NodeUpdateEvent))
	ProccessCommandMessageToDevice(ctx context.Context, devCmd types.DeviceCommandMessage)
	ProccessGetMessageToDevice(ctx context.Context, devCmd types.DeviceGetMessage)
	ProccessSetMessageToDevice(ctx context.Context, devCmd types.DeviceSetMessage)
	ProccessSetDeviceConfigMessage(ctx context.Context, devCmd types.DeviceConfigSetMessage)
	ProccessGetDeviceDescriptionMessage(ctx context.Context, devCmd types.DeviceExploreMessage)
	StartAsync(ctx context.Context)
	Stop()
}
