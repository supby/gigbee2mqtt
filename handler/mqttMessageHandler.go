package handler

import (
	"encoding/json"
	"strings"

	"github.com/supby/gigbee2mqtt/mqtt"
)

type MQTTMessageHandler struct {
	mqttClient *mqtt.Client
}

func CreateMQTTMessageHandler(mqttClient *mqtt.Client) *MQTTMessageHandler {
	ret := MQTTMessageHandler{
		mqttClient: mqttClient,
	}

	mqttClient.Subscribe(ret.mqqtMessage)

	return &ret
}

func (h *MQTTMessageHandler) mqqtMessage(topic string, message []byte) {

	var devMsg mqtt.DeviceMessage
	json.Unmarshal(message, &devMsg)

	topicParts := strings.Split(topic, "/")
	if len(topicParts) < 3 {
		return
	}
}
