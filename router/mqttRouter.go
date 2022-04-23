package router

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
)

const (
	MQTT_DEVICE_SET  = "set"
	MQTT_DEVICE_GET  = "get"
	MQTT_GET_DEVICES = "get_devices"
	MQTT_DEVICES     = "devices"
	MQTT_GATEWAY     = "gateway"
)

type mqttRouter struct {
	mqttClient           mqtt.MqttClient
	configurationService configuration.ConfigurationService
	onSetMessage         func(devCmd types.DeviceCommandMessage)
	onGetMessage         func(devCmd types.DeviceGetMessage)
	db                   db.DevicesRepo
}

func NewMQTTRouter(
	configurationService configuration.ConfigurationService,
	mqttClient mqtt.MqttClient,
	db db.DevicesRepo) MQTTRouter {
	ret := mqttRouter{
		mqttClient:           mqttClient,
		configurationService: configurationService,
		db:                   db,
	}

	mqttClient.Subscribe(ret.mqttMessage)

	return &ret
}

func (h *mqttRouter) ProccessMessageFromDevice(devMsg mqtt.DeviceMessage) {
	jsonData, err := json.Marshal(devMsg)
	if err != nil {
		log.Printf("[MQTT Router] Error Marshal DeviceMessage: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("0x%x", devMsg.IEEEAddress), jsonData)
}

func (h *mqttRouter) SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage)) {
	h.onSetMessage = callback
}

func (h *mqttRouter) SubscribeOnGetMessage(callback func(devCmd types.DeviceGetMessage)) {
	h.onGetMessage = callback
}

func (h *mqttRouter) mqttMessage(topic string, message []byte) {
	topicParts := strings.Split(topic, "/")
	if len(topicParts) < 3 {
		return
	}

	if topicParts[1] == MQTT_GATEWAY {
		h.handleGatewayMessage(topicParts[2], message)
		return
	}

	h.handleDeviceMessage(topicParts[1], topicParts[2], message)
}

func (h *mqttRouter) handleGatewayMessage(command string, message []byte) {
	if command == MQTT_GET_DEVICES {
		h.publishDevicesList()
	}
}

func (h *mqttRouter) publishDevicesList() {
	jsonData, err := json.Marshal(h.db.GetNodes())
	if err != nil {
		log.Printf("[MQTT Router] Error Marshal Devices list: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("%v/%v", MQTT_GATEWAY, MQTT_DEVICES), jsonData)
}

func (h *mqttRouter) handleDeviceMessage(deviceAddrStr string, command string, message []byte) {

	deviceAddr, err := strconv.ParseUint(strings.Replace(deviceAddrStr, "0x", "", -1), 16, 64)
	if err != nil {
		log.Printf("[MQTT Router] Error parsing device address as uint64: %v\n", err)
	}

	if command == MQTT_DEVICE_GET {
		h.handleDeviceGetCommand(deviceAddr, message)
	}

	if command == MQTT_DEVICE_SET {
		h.handleDeviceSetCommand(deviceAddr, message)
	}
}

func (h *mqttRouter) handleDeviceGetCommand(deviceAddr uint64, message []byte) {
	var devMsg mqtt.DeviceGetMessage
	err := json.Unmarshal(message, &devMsg)
	if err != nil {
		log.Printf("[MQTT Router] Error unmarshal GET message: %v\n", err)
		return
	}

	log.Printf("[MQTT Router] GET message received. Device:%v", deviceAddr)

	if h.onGetMessage != nil {
		h.onGetMessage(types.DeviceGetMessage{
			IEEEAddress: deviceAddr,
			ClusterID:   devMsg.ClusterID,
			Endpoint:    devMsg.Endpoint,
		})
	}
}

func (h *mqttRouter) handleDeviceSetCommand(deviceAddr uint64, message []byte) {
	var devMsg mqtt.DeviceSetMessage
	err := json.Unmarshal(message, &devMsg)
	if err != nil {
		log.Printf("[MQTT Router] Error unmarshal SET message: %v\n", err)
		return
	}

	log.Printf("[MQTT Router] SET message received. Device:%v, ClusterID:%v, CommandID:%v", deviceAddr, devMsg.ClusterID, devMsg.CommandIdentifier)

	if h.onSetMessage != nil {
		h.onSetMessage(types.DeviceCommandMessage{
			IEEEAddress:       deviceAddr,
			ClusterID:         devMsg.ClusterID,
			Endpoint:          devMsg.Endpoint,
			CommandIdentifier: devMsg.CommandIdentifier,
			CommandData:       devMsg.CommandData,
		})
	}

}
