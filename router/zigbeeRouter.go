package router

import (
	"context"
	"log"
	"time"

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
	zstack                     *zstack.ZStack
	configuration              *configuration.Configuration
	zclCommandRegistry         *zcl.CommandRegistry
	zclDefService              zcldef.ZCLDefService
	database                   db.DevicesRepo
	onDeviceMessage            func(devMsg mqtt.DeviceMessage)
	onDeviceDescriptionMessage func(devMsg mqtt.DeviceDescriptionMessage)
}

func (mh *zigbeeRouter) SubscribeOnDeviceMessage(callback func(devMsg mqtt.DeviceMessage)) {
	mh.onDeviceMessage = callback
}

func (mh *zigbeeRouter) SubscribeOnDeviceDescription(callback func(devMsg mqtt.DeviceDescriptionMessage)) {
	mh.onDeviceDescriptionMessage = callback
}

func (mh *zigbeeRouter) ProccessSetDeviceConfigMessage(ctx context.Context, devCmd types.DeviceConfigSetMessage) {
	if devCmd.PermitJoin == mh.configuration.PermitJoin {
		return
	}

	if devCmd.PermitJoin {
		err := mh.zstack.PermitJoin(ctx, true)
		if err != nil {
			log.Printf("[Device Router] Error PermitJoin, %v\n", err)
		}
	} else {
		err := mh.zstack.DenyJoin(ctx)
		if err != nil {
			log.Printf("[Device Router] Error DenyJoin to true, %v\n", err)
		}

	}
}

func (mh *zigbeeRouter) ProccessGetDeviceDescriptionMessage(ctx context.Context, devCmd types.DeviceExploreMessage) {
	log.Printf("[Device Router] Quering description of node %v\n", devCmd.IEEEAddress)

	ret := mqtt.DeviceDescriptionMessage{
		IEEEAddress: devCmd.IEEEAddress,
		Endpoints:   make([]mqtt.EndpointDescription, 0),
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	descriptor, err := mh.zstack.QueryNodeDescription(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress))
	if err != nil {
		log.Printf("[Device Router] Failed to get node descriptor: %v\n", err)
		return
	}

	ret.LogicalType = uint8(descriptor.LogicalType)
	ret.ManufacturerCode = uint16(descriptor.ManufacturerCode)

	endpoints, err := mh.zstack.QueryNodeEndpoints(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress))
	if err != nil {
		log.Printf("[Device Router] Failed to get node endpoints: %v\n", err)
		return
	}

	for _, endpoint := range endpoints {
		endpointDes, err := mh.zstack.QueryNodeEndpointDescription(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress), endpoint)

		if err != nil {
			log.Printf("[Device Router] Failed to get node endpoint description: %v / %d\n", err, endpoint)
			continue
		}

		newEl := mqtt.EndpointDescription{
			Endpoint:       uint8(endpointDes.Endpoint),
			ProfileID:      uint16(endpointDes.ProfileID),
			DeviceID:       endpointDes.DeviceID,
			DeviceVersion:  endpointDes.DeviceVersion,
			InClusterList:  make([]uint16, len(endpointDes.InClusterList)),
			OutClusterList: make([]uint16, len(endpointDes.OutClusterList)),
		}

		for i, v := range endpointDes.InClusterList {
			newEl.InClusterList[i] = uint16(v)
		}

		for i, v := range endpointDes.OutClusterList {
			newEl.OutClusterList[i] = uint16(v)
		}

		ret.Endpoints = append(ret.Endpoints, newEl)
	}
}

func (mh *zigbeeRouter) ProccessGetMessageToDevice(ctx context.Context, devCmd types.DeviceGetMessage) {

	attributeIds := make([]zcl.AttributeID, 0)
	for _, attr := range devCmd.Attributes {
		attributeIds = append(attributeIds, zcl.AttributeID(attr))
	}

	message := zcl.Message{
		FrameType:           zcl.FrameGlobal,
		Direction:           zcl.ClientToServer,
		TransactionSequence: 1, // TODO: do something with this
		Manufacturer:        zigbee.NoManufacturer,
		ClusterID:           zigbee.ClusterID(devCmd.ClusterID),
		SourceEndpoint:      zigbee.Endpoint(0x01),
		DestinationEndpoint: zigbee.Endpoint(devCmd.Endpoint),
		CommandIdentifier:   global.ReadAttributesID,
		Command: &global.ReadAttributes{
			Identifier: attributeIds,
		},
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

	log.Printf("[ProcessIncomingMessage] Incomming command of type (%T) is received. ClusterId=%v, SourceEndpoint=%v\n",
		message.Command, message.ClusterID, message.SourceEndpoint)

	switch cmd := message.Command.(type) {
	case *global.ReportAttributes:
		mh.processReportAttributes(msg, cmd)
	case *global.DefaultResponse:
		mh.processDefaultResponse(msg, cmd)
	case *global.ReadAttributesResponse:
		mh.processReadAttributesResponse(msg, cmd)
	}
}

func (mh *zigbeeRouter) processReadAttributesResponse(msg zigbee.IncomingMessage, cmd *global.ReadAttributesResponse) {
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
