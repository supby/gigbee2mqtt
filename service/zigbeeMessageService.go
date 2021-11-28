package service

import (
	"context"
	"log"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/onoff"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"
	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
	"github.com/supby/gigbee2mqtt/zcldef"
)

type ZigbeeMessageService struct {
	zstack             *zstack.ZStack
	configuration      *configuration.Configuration
	zclCommandRegistry *zcl.CommandRegistry
	zclDefService      zcldef.ZCLDefService
	database           db.DevicesRepo
	onAttributesReport func(devMsg mqtt.DeviceAttributesReportMessage)
}

func (mh *ZigbeeMessageService) SubscribeOnAttributesReport(callback func(devMsg mqtt.DeviceAttributesReportMessage)) {
	mh.onAttributesReport = callback
}

func (mh *ZigbeeMessageService) ProccessMessageToDevice(devCmd types.DeviceCommandMessage) {

	appMsg, err := mh.zclCommandRegistry.Marshal(zcl.Message{
		FrameType:           zcl.FrameLocal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: 1,
		Manufacturer:        zigbee.NoManufacturer,
		ClusterID:           zigbee.ClusterID(devCmd.ClusterID),
		SourceEndpoint:      zigbee.Endpoint(0x01),
		DestinationEndpoint: zigbee.Endpoint(devCmd.Endpoint),
		CommandIdentifier:   zcl.CommandIdentifier(devCmd.CommandIdentifier),
		Command:             &onoff.Toggle{},
		//Command:             &devCmd.CommandData,
	})

	// appMsg, err := mh.zclCommandRegistry.Marshal(zcl.Message{
	// 	FrameType:           zcl.FrameLocal,
	// 	Direction:           zcl.ClientToServer,
	// 	TransactionSequence: 1,
	// 	Manufacturer:        zigbee.NoManufacturer,
	// 	ClusterID:           zigbee.ClusterID(8),
	// 	SourceEndpoint:      zigbee.Endpoint(0x01),
	// 	DestinationEndpoint: zigbee.Endpoint(devCmd.Endpoint),
	// 	CommandIdentifier:   level.MoveToLevelWithOnOffId,
	// 	Command: &level.MoveToLevelWithOnOff{
	// 		Level:          108,
	// 		TransitionTime: 1,
	// 	},
	// })

	if err != nil {
		log.Printf("Error Marshal zcl message: %v\n", err)
		return
	}

	err = mh.zstack.SendApplicationMessageToNode(context.Background(), zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, true)
	if err != nil {
		log.Printf("Error sending message: %v\n", err)
		return
	}
}

func saveNodeDB(znode zigbee.Node, dbObj db.DevicesRepo) {
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

func (mh *ZigbeeMessageService) processNodeJoin(e zigbee.NodeJoinEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeMessageService) processNodeUpdate(e zigbee.NodeUpdateEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeMessageService) processIncomingMessage(e zigbee.NodeIncomingMessageEvent) {
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
		mh.processReportAttributes(msg, cmd)
	}
}

func (mh *ZigbeeMessageService) processReportAttributes(msg zigbee.IncomingMessage, cmd *global.ReportAttributes) {
	clusterDef := mh.zclDefService.GetById(uint16(msg.ApplicationMessage.ClusterID))

	mqttMessage := mqtt.DeviceAttributesReportMessage{
		IEEEAddress: uint64(msg.SourceAddress.IEEEAddress),
		LinkQuality: msg.LinkQuality,
		ClusterAttributes: mqtt.ClusterAttributesData{
			ID:         clusterDef.ID,
			Name:       clusterDef.Name,
			Attributes: make(map[string]interface{}),
		},
	}

	for _, r := range cmd.Records {
		log.Printf("AttrId: %v, DataType: %v, Value (%T): %v\n", r.Identifier, r.DataTypeValue.DataType, r.DataTypeValue.Value, r.DataTypeValue.Value)

		attrDef := clusterDef.Attributes[uint16(r.Identifier)]
		mqttMessage.ClusterAttributes.Attributes[attrDef.Name] = r.DataTypeValue.Value
	}

	if mh.onAttributesReport != nil {
		mh.onAttributesReport(mqttMessage)
	}
}

func CreateZigbeeMessageService(
	z *zstack.ZStack,
	zclCommandRegistry *zcl.CommandRegistry,
	zclDefService zcldef.ZCLDefService,
	database db.DevicesRepo,
	cfg *configuration.Configuration) *ZigbeeMessageService {
	ret := ZigbeeMessageService{
		zstack:             z,
		configuration:      cfg,
		zclCommandRegistry: zclCommandRegistry,
		zclDefService:      zclDefService,
		database:           database,
	}

	return &ret
}

func (mh *ZigbeeMessageService) StartAsync(ctx context.Context) {
	go mh.startEventLoop(ctx)
}

func (mh *ZigbeeMessageService) startEventLoop(ctx context.Context) {
	log.Println("Start event loop ====")
	for {
		event, err := mh.zstack.ReadEvent(ctx)

		if err != nil {
			log.Printf("Error read event: %v\n", err)
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			log.Printf("join: %v\n", e.Node)
			//mh.exploreDevice(ctx, e.Node)
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

// func (mh *ZigbeeMessageService) exploreDevice(ctx context.Context, node zigbee.Node) {
// 	log.Printf("node %v: querying\n", node.IEEEAddress)

// 	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
// 	defer cancel()

// 	descriptor, err := mh.zstack.QueryNodeDescription(ctx, node.IEEEAddress)

// 	if err != nil {
// 		log.Printf("failed to get node descriptor: %v\n", err)
// 		return
// 	}

// 	log.Printf("node %v: descriptor: %+v\n", node.IEEEAddress, descriptor)

// 	endpoints, err := mh.zstack.QueryNodeEndpoints(ctx, node.IEEEAddress)

// 	if err != nil {
// 		log.Printf("failed to get node endpoints: %v\n", err)
// 		return
// 	}

// 	log.Printf("node %v: endpoints: %+v\n", node.IEEEAddress, endpoints)

// 	for _, endpoint := range endpoints {
// 		endpointDes, err := mh.zstack.QueryNodeEndpointDescription(ctx, node.IEEEAddress, endpoint)

// 		if err != nil {
// 			log.Printf("failed to get node endpoint description: %v / %d\n", err, endpoint)
// 		} else {
// 			log.Printf("node %v: endpoint: %d desc: %+v", node.IEEEAddress, endpoint, endpointDes)
// 		}
// 	}
// }
