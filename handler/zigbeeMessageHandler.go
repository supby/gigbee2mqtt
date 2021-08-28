package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/zcldef"
)

type ZigbeeMessageHandler struct {
	configuration      *configuration.Configuration
	zclCommandRegistry *zcl.CommandRegistry
	zclDefMap          *zcldef.ZCLDefMap
	mqttClient         *mqtt.Client
	database           *db.DB
}

func saveNodeDB(znode zigbee.Node, dbObj *db.DB) {
	dbNode := db.Node{
		IEEEAddress:    uint64(znode.IEEEAddress),
		NetworkAddress: uint16(znode.NetworkAddress),
		LogicalType:    uint8(znode.LogicalType),
		LQI:            znode.LQI,
		Depth:          znode.Depth,
		LastDiscovered: znode.LastDiscovered,
		LastReceived:   znode.LastReceived,
	}

	dbObj.SaveNode(dbNode)
}

func (mh *ZigbeeMessageHandler) ProcessNodeJoin(e zigbee.NodeJoinEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeMessageHandler) ProcessNodeUpdate(e zigbee.NodeUpdateEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeMessageHandler) ProcessIncomingMessage(e zigbee.NodeIncomingMessageEvent) {
	go saveNodeDB(e.Node, mh.database)
	msg := e.IncomingMessage
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
			log.Printf("AttrId: %v, DataType: %v, Value (%T): %v\n", r.Identifier, r.DataTypeValue.DataType, r.DataTypeValue.Value, r.DataTypeValue.Value)

			attrDef := clusterDef.Attributes[uint16(r.Identifier)]
			mqttMessage.Cluster.Attributes[attrDef.Name] = r.DataTypeValue.Value
		}

		jsonData, _ := json.Marshal(mqttMessage)
		mh.mqttClient.Publish(fmt.Sprintf("%v/%v", mh.configuration.MqttConfiguration.Topic, mqttMessage.IEEEAddress), jsonData)

	}
}

func CreateZigbeeMessageHandler(
	zclCommandRegistry *zcl.CommandRegistry,
	zclDefMap *zcldef.ZCLDefMap,
	mqttClient *mqtt.Client,
	database *db.DB,
	cfg *configuration.Configuration) *ZigbeeMessageHandler {
	ret := ZigbeeMessageHandler{
		configuration:      cfg,
		zclCommandRegistry: zclCommandRegistry,
		zclDefMap:          zclDefMap,
		mqttClient:         mqttClient,
		database:           database,
	}

	return &ret
}
