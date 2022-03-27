package router

import (
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
)

type MQTTRouter interface {
	ProccessMessageFromDevice(devMsg mqtt.DeviceMessage)
	SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage))
}
