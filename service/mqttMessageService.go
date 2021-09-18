package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/mqtt"
)

const (
	MQTT_SET = "set"
	MQTT_GET = "get"
)

type MQTTMessageService struct {
	mqttClient    *mqtt.MqttClient
	configuration *configuration.Configuration
	onSetMessage  func(devMsg mqtt.DeviceSetMessage)
}

func CreateMQTTMessageService(
	configuration *configuration.Configuration,
	mqttClient *mqtt.MqttClient) *MQTTMessageService {
	ret := MQTTMessageService{
		mqttClient:    mqttClient,
		configuration: configuration,
	}

	mqttClient.Subscribe(ret.mqqtMessage)

	return &ret
}

func (h *MQTTMessageService) ProccessMessageFromDevice(devMsg mqtt.DeviceAttributesReportMessage) {
	jsonData, err := json.Marshal(devMsg)
	if err != nil {
		log.Printf("Error Marshal Set DeviceAttributesReportMessage: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("%v/%v", h.configuration.MqttConfiguration.Topic, devMsg.IEEEAddress), jsonData)
}

func (h *MQTTMessageService) SubscribeOnSetMessage(callback func(devMsg mqtt.DeviceSetMessage)) {
	h.onSetMessage = callback
}

func (h *MQTTMessageService) mqqtMessage(topic string, message []byte) {
	topicParts := strings.Split(topic, "/")
	if len(topicParts) < 3 {
		return
	}

	if topicParts[2] == MQTT_GET {
		h.handleGetCommand(topicParts[1], message)
	}

	if topicParts[2] == MQTT_SET {
		h.handleSetCommand(topicParts[1], message)
	}
}

func (h *MQTTMessageService) handleGetCommand(deviceAddr string, message []byte) {

}

func (h *MQTTMessageService) handleSetCommand(deviceAddr string, message []byte) {
	var devMsg mqtt.DeviceSetMessage
	err := json.Unmarshal(message, &devMsg)
	if err != nil {
		log.Printf("Error unmarshal Set message: %v\n", err)
		return
	}

	if h.onSetMessage != nil {
		h.onSetMessage(devMsg)
	}

}
