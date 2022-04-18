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

type zigbeeRouter struct {
	zstack             *zstack.ZStack
	configuration      *configuration.Configuration
	zclCommandRegistry *zcl.CommandRegistry
	zclDefService      zcldef.ZCLDefService
	database           db.DevicesRepo
	onDeviceMessage    func(devMsg mqtt.DeviceMessage)
}

func (mh *zigbeeRouter) SubscribeOnDeviceMessage(callback func(devMsg mqtt.DeviceMessage)) {
	mh.onDeviceMessage = callback
}

func (mh *zigbeeRouter) ProccessGetMessageToDevice(ctx context.Context, devCmd types.DeviceGetMessage) {
	message := zcl.Message{
		FrameType:           zcl.FrameGlobal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: 1, // TODO: do something with this
		Manufacturer:        zigbee.NoManufacturer,
		ClusterID:           zigbee.ClusterID(devCmd.ClusterID),
		SourceEndpoint:      zigbee.Endpoint(0x01),
		DestinationEndpoint: zigbee.Endpoint(devCmd.Endpoint),
		CommandIdentifier:   global.ReadAttributesID,
		Command:             &global.ReadAttributes{},
	}

	appMsg, err := mh.zclCommandRegistry.Marshal(message)
	if err != nil {
		log.Printf("[ProccessGetMessageToDevice] Error Marshal zcl message: %v\n", err)
		return
	}

	err = mh.zstack.SendApplicationMessageToNode(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, false)
	if err != nil {
		log.Printf("[ProccessGetMessageToDevice] Error sending message: %v\n", err)
		return
	}

	log.Printf("[ProccessMessageToDevice] Message (ClusterID: %v, Command: %v) is sent to %v device\n", message.ClusterID, message.CommandIdentifier, devCmd.IEEEAddress)
}

func (mh *zigbeeRouter) ProccessMessageToDevice(ctx context.Context, devCmd types.DeviceCommandMessage) {

	message := zcl.Message{
		FrameType:           zcl.FrameLocal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: 1, // TODO: do something with this
		Manufacturer:        zigbee.NoManufacturer,
		ClusterID:           zigbee.ClusterID(devCmd.ClusterID),
		SourceEndpoint:      zigbee.Endpoint(0x01),
		DestinationEndpoint: zigbee.Endpoint(devCmd.Endpoint),
		CommandIdentifier:   zcl.CommandIdentifier(devCmd.CommandIdentifier),
	}

	command, err := mh.zclCommandRegistry.GetLocalCommand(message.ClusterID, message.Manufacturer, message.Direction, message.CommandIdentifier)
	if err != nil {
		log.Printf("[ProccessMessageToDevice] Error Local command for ClusterID: %v, Manufacturer: %v, Direction: %v, CommandIdentifier: %v. Error: %v \n",
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
		log.Printf("[ProccessMessageToDevice] Error Marshal zcl message: %v\n", err)
		return
	}

	// timeoutCtx, timeoutCancel := context.WithTimeout(ctx, time.Minute)
	// defer timeoutCancel()

	//err = mh.zstack.SendApplicationMessageToNode(timeoutCtx, zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, true)
	err = mh.zstack.SendApplicationMessageToNode(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, false)
	if err != nil {
		log.Printf("[ProccessMessageToDevice] Error sending message: %v\n", err)
		return
	}

	log.Printf("[ProccessMessageToDevice] Message (ClusterID: %v, Command: %v) is sent to %v device\n", message.ClusterID, message.CommandIdentifier, devCmd.IEEEAddress)
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

func (mh *zigbeeRouter) processNodeJoin(e zigbee.NodeJoinEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *zigbeeRouter) processNodeLeave(e zigbee.NodeLeaveEvent) {

}

func (mh *zigbeeRouter) processNodeUpdate(e zigbee.NodeUpdateEvent) {
	saveNodeDB(e.Node, mh.database)
}

func (mh *zigbeeRouter) processIncomingMessage(e zigbee.NodeIncomingMessageEvent) {
	go saveNodeDB(e.Node, mh.database)
	msg := e.IncomingMessage
	message, err := mh.zclCommandRegistry.Unmarshal(msg.ApplicationMessage)
	if err != nil {
		log.Printf("[ProcessIncomingMessage] Error parse incomming message: %v\n", err)
		return
	}

	log.Printf("[ProcessIncomingMessage] Incomming command of type (%T) is received. ClusterId is %v\n", message.Command, message.ClusterID)

	switch cmd := message.Command.(type) {
	case *global.ReportAttributes:
		mh.processReportAttributes(msg, cmd)
	case *global.DefaultResponse:
		mh.processDefaultResponse(msg, cmd)
	}
}

func (mh *zigbeeRouter) processReportAttributes(msg zigbee.IncomingMessage, cmd *global.ReportAttributes) {
	clusterDef := mh.zclDefService.GetById(uint16(msg.ApplicationMessage.ClusterID))

	mqttMessage := mqtt.DeviceMessage{
		IEEEAddress: uint64(msg.SourceAddress.IEEEAddress),
		LinkQuality: msg.LinkQuality,
	}

	deviceMessage := mqtt.DeviceAttributesReportMessage{
		ClusterID:         clusterDef.ID,
		ClusterName:       clusterDef.Name,
		ClusterAttributes: make(map[string]interface{}),
	}

	for _, r := range cmd.Records {
		attrDef := clusterDef.Attributes[uint16(r.Identifier)]
		deviceMessage.ClusterAttributes[attrDef.Name] = r.DataTypeValue.Value
	}

	mqttMessage.Message = deviceMessage

	if mh.onDeviceMessage != nil {
		mh.onDeviceMessage(mqttMessage)
	}
}

func (mh *zigbeeRouter) processDefaultResponse(msg zigbee.IncomingMessage, cmd *global.DefaultResponse) {
	mqttMessage := mqtt.DeviceMessage{
		IEEEAddress: uint64(msg.SourceAddress.IEEEAddress),
		LinkQuality: msg.LinkQuality,
		Message: mqtt.DeviceDefaultResponseMessage{
			ClusterID:         uint16(msg.ApplicationMessage.ClusterID),
			CommandIdentifier: cmd.CommandIdentifier,
			Status:            cmd.Status,
		},
	}

	if mh.onDeviceMessage != nil {
		mh.onDeviceMessage(mqttMessage)
	}
}

func NewZigbeeRouter(
	z *zstack.ZStack,
	zclCommandRegistry *zcl.CommandRegistry,
	zclDefService zcldef.ZCLDefService,
	database db.DevicesRepo,
	cfg *configuration.Configuration) ZigbeeRouter {
	ret := zigbeeRouter{
		zstack:             z,
		configuration:      cfg,
		zclCommandRegistry: zclCommandRegistry,
		zclDefService:      zclDefService,
		database:           database,
	}

	return &ret
}

func (mh *zigbeeRouter) StartAsync(ctx context.Context) {
	go mh.startEventLoop(ctx)
}

func (mh *zigbeeRouter) startEventLoop(ctx context.Context) {
	log.Println("[Event loop] Start event")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		event, err := mh.zstack.ReadEvent(ctx)
		if err != nil {
			log.Printf("[Event loop] Error read event: %v\n", err)
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			log.Printf("[Event loop] Node join: %v\n", e)
			go mh.processNodeJoin(e)
		case zigbee.NodeLeaveEvent:
			log.Printf("[Event loop] Node leave: %v\n", e)
			go mh.processNodeLeave(e)
		case zigbee.NodeUpdateEvent:
			log.Printf("[Event loop] Node update: %v\n", e)
			go mh.processNodeUpdate(e)
		case zigbee.NodeIncomingMessageEvent:
			log.Printf("[Event loop] Node message: %v\n", e)
			go mh.processIncomingMessage(e)
		}
	}
}
