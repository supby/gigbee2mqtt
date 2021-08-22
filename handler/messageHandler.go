package handler

import (
	"encoding/json"
	"fmt"
	"log"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/zcldef"
)

type MessageHandler struct {
	configuration      *configuration.Configuration
	zclCommandRegistry *zcl.CommandRegistry
	zclDefMap          *zcldef.ZCLDefMap
	mqttClient         mqttlib.Client
}

func (mh *MessageHandler) ProcessIncomingMessage(msg zigbee.IncomingMessage) {
	message, err := mh.zclCommandRegistry.Unmarshal(msg.ApplicationMessage)
	if err != nil {
		log.Printf("Error parse incomming message: %v\n", err)
		return
	}

	log.Printf("Incomming command of type (%T) is received. ClusterId is %v\n", message.Command, message.ClusterID)

	switch cmd := message.Command.(type) {
	case *global.ReportAttributes:
		clusterDef := (*mh.zclDefMap)[uint16(msg.ApplicationMessage.ClusterID)]

		mqttMessage := mqtt.DeviceMessage{
			IEEEAddress: uint64(msg.SourceAddress.IEEEAddress),
			LinkQuality: msg.LinkQuality,
			Cluster: mqtt.ClusterData{
				ID:         clusterDef.ID,
				Name:       clusterDef.Name,
				Attributes: make(map[string]interface{}),
			},
		}

		for _, r := range cmd.Records {
			// AttrId: 2, DataType: 33, Value: 0
			log.Printf("AttrId: %v, DataType: %v, Value (%T): %v\n", r.Identifier, r.DataTypeValue.DataType, r.DataTypeValue.Value, r.DataTypeValue.Value)

			attrDef := clusterDef.Attributes[uint16(r.Identifier)]
			mqttMessage.Cluster.Attributes[attrDef.Name] = r.DataTypeValue.Value
		}

		jsonData, _ := json.Marshal(mqttMessage)
		mh.mqttClient.Publish(fmt.Sprintf("%v/%v", mh.configuration.MqttConfiguration.Topic, mqttMessage.IEEEAddress), 0, false, jsonData)

	}
}

func Create(
	zclCommandRegistry *zcl.CommandRegistry,
	zclDefMap *zcldef.ZCLDefMap,
	mqttClient mqttlib.Client,
	cfg *configuration.Configuration) *MessageHandler {
	ret := MessageHandler{
		configuration:      cfg,
		zclCommandRegistry: zclCommandRegistry,
		zclDefMap:          zclDefMap,
		mqttClient:         mqttClient,
	}

	return &ret
}
