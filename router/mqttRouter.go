package router

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
)

const (
	MQTT_SET = "set"
	MQTT_GET = "get"
)

type MQTTRouter struct {
	mqttClient    mqtt.MqttClient
	configuration *configuration.Configuration
	onSetMessage  func(devCmd types.DeviceCommandMessage)
}

func NewMQTTRouter(
	configuration *configuration.Configuration,
	mqttClient mqtt.MqttClient) *MQTTRouter {
	ret := MQTTRouter{
		mqttClient:    mqttClient,
		configuration: configuration,
	}

	mqttClient.Subscribe(ret.mqttMessage)

	return &ret
}

func (h *MQTTRouter) ProccessMessageFromDevice(devMsg mqtt.DeviceMessage) {
	jsonData, err := json.Marshal(devMsg)
	if err != nil {
		log.Printf("[MQTT Router] Error Marshal DeviceMessage: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("0x%x", devMsg.IEEEAddress), jsonData)
}

func (h *MQTTRouter) SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage)) {
	h.onSetMessage = callback
}

func (h *MQTTRouter) mqttMessage(topic string, message []byte) {
	topicParts := strings.Split(topic, "/")
	if len(topicParts) < 3 {
		return
	}

	deviceAddr, err := strconv.ParseUint(strings.Replace(topicParts[1], "0x", "", -1), 16, 64)
	if err != nil {
		log.Printf("[MQTT Router] Error parsing device address as uint64: %v\n", err)
	}

	if topicParts[2] == MQTT_GET {
		h.handleGetCommand(deviceAddr, message)
	}

	if topicParts[2] == MQTT_SET {
		h.handleSetCommand(deviceAddr, message)
	}
}

func (h *MQTTRouter) handleGetCommand(deviceAddr uint64, message []byte) {

}

func (h *MQTTRouter) handleSetCommand(deviceAddr uint64, message []byte) {
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
