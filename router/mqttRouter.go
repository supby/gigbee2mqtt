package router

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/logger"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
)

const (
	MQTT_DEVICE_SET     = "set"
	MQTT_DEVICE_GET     = "get"
	MQTT_DEVICE_EXPLORE = "explore"
	MQTT_GET_DEVICES    = "get_devices"
	MQTT_GET_CONFIG     = "get_config"
	MQTT_SET_CONFIG     = "set_config"
	MQTT_DEVICES        = "devices"
	MQTT_CONFIG         = "config"
	MQTT_GATEWAY        = "gateway"
)

type mqttRouter struct {
	mqttClient               mqtt.MqttClient
	configurationService     configuration.ConfigurationService
	onSetMessage             func(devCmd types.DeviceCommandMessage)
	onGetMessage             func(devCmd types.DeviceGetMessage)
	onExploreMessage         func(devCmd types.DeviceExploreMessage)
	onSetDeviceConfigMessage func(devCmd types.DeviceConfigSetMessage)
	db                       db.DevicesRepo
	logger                   logger.Logger
}

func NewMQTTRouter(
	configurationService configuration.ConfigurationService,
	mqttClient mqtt.MqttClient,
	db db.DevicesRepo) MQTTRouter {
	ret := mqttRouter{
		mqttClient:           mqttClient,
		configurationService: configurationService,
		db:                   db,
		logger:               logger.GetLogger("[MQTT Router]"),
	}

	mqttClient.Subscribe(ret.mqttMessage)

	return &ret
}

func (h *mqttRouter) ProccessMessageFromDevice(devMsg mqtt.DeviceMessage) {
	jsonData, err := json.Marshal(devMsg)
	if err != nil {
		h.logger.Log("Error Marshal DeviceMessage: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("0x%x", devMsg.IEEEAddress), jsonData)
}

func (h *mqttRouter) ProccessDeviceDescriptionMessage(devDscMsg mqtt.DeviceDescriptionMessage) {
	jsonData, err := json.Marshal(devDscMsg)
	if err != nil {
		h.logger.Log("Error Marshal DeviceDescriptionMessage: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("0x%x/description", devDscMsg.IEEEAddress), jsonData)
}

func (h *mqttRouter) SubscribeOnSetMessage(callback func(devCmd types.DeviceCommandMessage)) {
	h.onSetMessage = callback
}

func (h *mqttRouter) SubscribeOnGetMessage(callback func(devCmd types.DeviceGetMessage)) {
	h.onGetMessage = callback
}

func (h *mqttRouter) SubscribeOnExploreMessage(callback func(devCmd types.DeviceExploreMessage)) {
	h.onExploreMessage = callback
}

func (h *mqttRouter) SubscribeOnSetDeviceConfigMessage(callback func(devCmd types.DeviceConfigSetMessage)) {
	h.onSetDeviceConfigMessage = callback
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
	if command == MQTT_GET_CONFIG {
		h.publishConfig()
	}
	if command == MQTT_SET_CONFIG {
		h.handleSetConfig(message)
	}
}

func (h *mqttRouter) publishConfig() {
	jsonData, err := json.Marshal(h.configurationService.GetConfiguration())
	if err != nil {
		h.logger.Log("Error Marshal Configuration: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("%v/%v", MQTT_GATEWAY, MQTT_CONFIG), jsonData)
}

func (h *mqttRouter) handleSetConfig(message []byte) {
	var mqttMsg mqtt.SetGatewayConfig
	err := json.Unmarshal(message, &mqttMsg)
	if err != nil {
		h.logger.Log("Error unmarshal Config SET message: %v\n", err)
		return
	}

	cfg := h.configurationService.GetConfiguration()
	cfg.PermitJoin = mqttMsg.PermitJoin

	err = h.configurationService.Update(cfg)
	if err != nil {
		h.logger.Log("Applying new configuration error: %v\n", err)
		return
	}

	if h.onSetDeviceConfigMessage != nil {
		h.onSetDeviceConfigMessage(types.DeviceConfigSetMessage{
			PermitJoin: mqttMsg.PermitJoin,
		})
	}
}

func (h *mqttRouter) publishDevicesList() {
	jsonData, err := json.Marshal(h.db.GetNodes())
	if err != nil {
		h.logger.Log("Error Marshal Devices list: %v\n", err)
		return
	}

	h.mqttClient.Publish(fmt.Sprintf("%v/%v", MQTT_GATEWAY, MQTT_DEVICES), jsonData)
}

func (h *mqttRouter) handleDeviceMessage(deviceAddrStr string, command string, message []byte) {

	deviceAddr, err := strconv.ParseUint(strings.Replace(deviceAddrStr, "0x", "", -1), 16, 64)
	if err != nil {
		h.logger.Log("Error parsing device address as uint64: %v\n", err)
	}

	if command == MQTT_DEVICE_GET {
		h.handleDeviceGetCommand(deviceAddr, message)
	}

	if command == MQTT_DEVICE_SET {
		h.handleDeviceSetCommand(deviceAddr, message)
	}

	if command == MQTT_DEVICE_EXPLORE {
		h.handleDeviceExploreCommand(deviceAddr, message)
	}
}

func (h *mqttRouter) handleDeviceExploreCommand(deviceAddr uint64, message []byte) {
	h.logger.Log("EXPLORE message received. Device: 0x%x", deviceAddr)

	if h.onGetMessage != nil {
		h.onExploreMessage(types.DeviceExploreMessage{
			IEEEAddress: deviceAddr,
		})
	}
}

func (h *mqttRouter) handleDeviceGetCommand(deviceAddr uint64, message []byte) {
	var devMsg mqtt.DeviceGetMessage
	err := json.Unmarshal(message, &devMsg)
	if err != nil {
		h.logger.Log("Error unmarshal GET message: %v\n", err)
		return
	}

	h.logger.Log("GET message received. Device:%v", deviceAddr)

	if h.onGetMessage != nil {
		h.onGetMessage(types.DeviceGetMessage{
			IEEEAddress: deviceAddr,
			ClusterID:   devMsg.ClusterID,
			Endpoint:    devMsg.Endpoint,
			Attributes:  devMsg.Attributes,
		})
	}
}

func (h *mqttRouter) handleDeviceSetCommand(deviceAddr uint64, message []byte) {
	var devMsg mqtt.DeviceSetMessage
	err := json.Unmarshal(message, &devMsg)
	if err != nil {
		h.logger.Log("Error unmarshal SET message: %v\n", err)
		return
	}

	h.logger.Log("SET message received. Device:%v, ClusterID:%v, CommandID:%v", deviceAddr, devMsg.ClusterID, devMsg.CommandIdentifier)

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
