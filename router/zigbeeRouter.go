package router

import (
	"context"
	"log"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"
	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/types"
	"github.com/supby/gigbee2mqtt/utils"
	"github.com/supby/gigbee2mqtt/zcldef"
)

type ZigbeeRouter struct {
	zstack             *zstack.ZStack
	configuration      *configuration.Configuration
	zclCommandRegistry *zcl.CommandRegistry
	zclDefService      zcldef.ZCLDefService
	database           db.DevicesRepo
	onAttributesReport func(devMsg mqtt.DeviceAttributesReportMessage)
	onDefaultResponse  func(devMsg mqtt.DeviceDefaultResponseMessage)
}

func (mh *ZigbeeRouter) SubscribeOnAttributesReport(callback func(devMsg mqtt.DeviceAttributesReportMessage)) {
	mh.onAttributesReport = callback
}

func (mh *ZigbeeRouter) ProccessMessageToDevice(ctx context.Context, devCmd types.DeviceCommandMessage) {

	message := zcl.Message{
		FrameType:           zcl.FrameLocal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: 1,
		Manufacturer:        zigbee.NoManufacturer,
		ClusterID:           zigbee.ClusterID(devCmd.ClusterID),
		SourceEndpoint:      zigbee.Endpoint(0x01),
		DestinationEndpoint: zigbee.Endpoint(devCmd.Endpoint),
		CommandIdentifier:   zcl.CommandIdentifier(devCmd.CommandIdentifier),
	}

	command, err := mh.zclCommandRegistry.GetLocalCommand(message.ClusterID, message.Manufacturer, message.Direction, message.CommandIdentifier)
	if err != nil {
		log.Printf("Error Local command for ClusterID: %v, Manufacturer: %v, Direction: %v, CommandIdentifier: %v. Error: %v \n",
			message.ClusterID,
			message.Manufacturer,
			message.Direction,
			message.CommandIdentifier,
			err)
		return
	}

	utils.SetStructProperties(devCmd.CommandData, command)

	message.Command = command

	appMsg, err := mh.zclCommandRegistry.Marshal(message)

	if err != nil {
		log.Printf("Error Marshal zcl message: %v\n", err)
		return
	}

	err = mh.zstack.SendApplicationMessageToNode(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, true)
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

func (mh *ZigbeeRouter) processNodeJoin(e zigbee.NodeJoinEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeRouter) processNodeUpdate(e zigbee.NodeUpdateEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *ZigbeeRouter) processIncomingMessage(e zigbee.NodeIncomingMessageEvent) {
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
	case *global.DefaultResponse:
		mh.processDefaultResponse(msg, cmd)
	}
}

func (mh *ZigbeeRouter) processReportAttributes(msg zigbee.IncomingMessage, cmd *global.ReportAttributes) {
	clusterDef := mh.zclDefService.GetById(uint16(msg.ApplicationMessage.ClusterID))

	mqttMessage := mqtt.DeviceAttributesReportMessage{
		IEEEAddress:       uint64(msg.SourceAddress.IEEEAddress),
		LinkQuality:       msg.LinkQuality,
		ClusterID:         clusterDef.ID,
		ClusterName:       clusterDef.Name,
		ClusterAttributes: make(map[string]interface{}),
	}

	for _, r := range cmd.Records {
		log.Printf("AttrId: %v, DataType: %v, Value (%T): %v\n", r.Identifier, r.DataTypeValue.DataType, r.DataTypeValue.Value, r.DataTypeValue.Value)

		attrDef := clusterDef.Attributes[uint16(r.Identifier)]
		mqttMessage.ClusterAttributes[attrDef.Name] = r.DataTypeValue.Value
	}

	if mh.onAttributesReport != nil {
		mh.onAttributesReport(mqttMessage)
	}
}

func (mh *ZigbeeRouter) processDefaultResponse(msg zigbee.IncomingMessage, cmd *global.DefaultResponse) {
	mqttMessage := mqtt.DeviceDefaultResponseMessage{
		IEEEAddress:       uint64(msg.SourceAddress.IEEEAddress),
		LinkQuality:       msg.LinkQuality,
		ClusterID:         uint16(msg.ApplicationMessage.ClusterID),
		CommandIdentifier: cmd.CommandIdentifier,
		Status:            cmd.Status,
	}

	if mh.onAttributesReport != nil {
		mh.onDefaultResponse(mqttMessage)
	}
}

func NewZigbeeRouter(
	z *zstack.ZStack,
	zclCommandRegistry *zcl.CommandRegistry,
	zclDefService zcldef.ZCLDefService,
	database db.DevicesRepo,
	cfg *configuration.Configuration) *ZigbeeRouter {
	ret := ZigbeeRouter{
		zstack:             z,
		configuration:      cfg,
		zclCommandRegistry: zclCommandRegistry,
		zclDefService:      zclDefService,
		database:           database,
	}

	return &ret
}

func (mh *ZigbeeRouter) StartAsync(ctx context.Context) {
	go mh.startEventLoop(ctx)
}

func (mh *ZigbeeRouter) startEventLoop(ctx context.Context) {
	log.Println("Start event loop ====")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

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
