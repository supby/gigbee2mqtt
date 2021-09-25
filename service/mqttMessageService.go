package service

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

type MQTTMessageService struct {
	mqttClient    mqtt.IMqttClient
	configuration *configuration.Configuration
	onSetMessage  func(devCmd types.DeviceCommandMessage)
}

func CreateMQTTMessageService(
	configuration *configuration.Configuration,
	mqttClient mqtt.IMqttClient) *MQTTMessageService {
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

func (h *MQTTMessageService) SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage)) {
	h.onSetMessage = callback
}

func (h *MQTTMessageService) mqqtMessage(topic string, message []byte) {
	topicParts := strings.Split(topic, "/")
	if len(topicParts) < 3 {
		return
	}

	deviceAddr, err := strconv.ParseUint(strings.Replace(topicParts[1], "0x", "", -1), 16, 64)
	if err != nil {
		log.Printf("Error parsing device address as uint64: %v\n", err)
	}

	if topicParts[2] == MQTT_GET {

		h.handleGetCommand(deviceAddr, message)
	}

	if topicParts[2] == MQTT_SET {
		h.handleSetCommand(deviceAddr, message)
	}
}

func (h *MQTTMessageService) handleGetCommand(deviceAddr uint64, message []byte) {

}

func (h *MQTTMessageService) handleSetCommand(deviceAddr uint64, message []byte) {
	var devMsg mqtt.DeviceSetMessage
	err := json.Unmarshal(message, &devMsg)
	if err != nil {
		log.Printf("Error unmarshal Set message: %v\n", err)
		return
	}

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
