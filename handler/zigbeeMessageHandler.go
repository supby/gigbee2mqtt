package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"
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

func (mh *ZigbeeMessageHandler) processNodeJoin(e zigbee.NodeJoinEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeMessageHandler) processNodeUpdate(e zigbee.NodeUpdateEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeMessageHandler) processIncomingMessage(e zigbee.NodeIncomingMessageEvent) {
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
	z *zstack.ZStack,
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

	ret.startEventLoop(z)

	return &ret
}

func (mh *ZigbeeMessageHandler) startEventLoop(z *zstack.ZStack) {
	log.Println("Start event loop ====")
	for {
		ctx := context.Background()
		event, err := z.ReadEvent(ctx)

		if err != nil {
			log.Printf("Error read event: %v\n", err)
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			log.Printf("join: %v\n", e.Node)
			exploreDevice(z, e.Node)
			go mh.processNodeJoin(e)
		case zigbee.NodeLeaveEvent:
			log.Printf("leave: %v\n", e.Node)
		case zigbee.NodeUpdateEvent:
			log.Printf("update: %v\n", e.Node)
			go mh.processNodeUpdate(e)
		case zigbee.NodeIncomingMessageEvent:
			log.Printf("message: %v\n", e)
			go mh.processIncomingMessage(e)
		}
	}
}

func exploreDevice(z *zstack.ZStack, node zigbee.Node) {
	log.Printf("node %v: querying\n", node.IEEEAddress)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	descriptor, err := z.QueryNodeDescription(ctx, node.IEEEAddress)

	if err != nil {
		log.Printf("failed to get node descriptor: %v\n", err)
		return
	}

	log.Printf("node %v: descriptor: %+v\n", node.IEEEAddress, descriptor)

	endpoints, err := z.QueryNodeEndpoints(ctx, node.IEEEAddress)

	if err != nil {
		log.Printf("failed to get node endpoints: %v\n", err)
		return
	}

	log.Printf("node %v: endpoints: %+v\n", node.IEEEAddress, endpoints)

	for _, endpoint := range endpoints {
		endpointDes, err := z.QueryNodeEndpointDescription(ctx, node.IEEEAddress, endpoint)

		if err != nil {
			log.Printf("failed to get node endpoint description: %v / %d\n", err, endpoint)
		} else {
			log.Printf("node %v: endpoint: %d desc: %+v", node.IEEEAddress, endpoint, endpointDes)
		}
	}
}
