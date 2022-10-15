package router

import (
	"context"
	"time"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"
	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/logger"
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
	database                   db.DeviceDB
	onDeviceMessage            func(devMsg mqtt.DeviceMessage)
	onDeviceDescriptionMessage func(devMsg mqtt.DeviceDescriptionMessage)
	onDeviceJoin               func(e zigbee.NodeJoinEvent)
	onDeviceLeave              func(e zigbee.NodeLeaveEvent)
	onDeviceUpdate             func(e zigbee.NodeUpdateEvent)
	logger                     logger.Logger
}

func (mh *zigbeeRouter) SubscribeOnDeviceMessage(callback func(devMsg mqtt.DeviceMessage)) {
	mh.onDeviceMessage = callback
}

func (mh *zigbeeRouter) SubscribeOnDeviceDescription(callback func(devMsg mqtt.DeviceDescriptionMessage)) {
	mh.onDeviceDescriptionMessage = callback
}

func (mh *zigbeeRouter) SubscribeOnDeviceJoin(cb func(e zigbee.NodeJoinEvent)) {
	mh.onDeviceJoin = cb
}

func (mh *zigbeeRouter) SubscribeOnDeviceLeave(cb func(e zigbee.NodeLeaveEvent)) {
	mh.onDeviceLeave = cb
}

func (mh *zigbeeRouter) SubscribeOnDeviceUpdate(cb func(e zigbee.NodeUpdateEvent)) {
	mh.onDeviceUpdate = cb
}

func (mh *zigbeeRouter) ProccessSetDeviceConfigMessage(ctx context.Context, devCmd types.DeviceConfigSetMessage) {
	if devCmd.PermitJoin == mh.configuration.PermitJoin {
		return
	}

	if devCmd.PermitJoin {
		err := mh.zstack.PermitJoin(ctx, true)
		if err != nil {
			mh.logger.Log("Error PermitJoin, %v\n", err)
		}
	} else {
		err := mh.zstack.DenyJoin(ctx)
		if err != nil {
			mh.logger.Log("Error DenyJoin to true, %v\n", err)
		}

	}
}

func (mh *zigbeeRouter) ProccessGetDeviceDescriptionMessage(ctx context.Context, devCmd types.DeviceExploreMessage) {
	mh.logger.Log("Quering description of node 0x%x\n", devCmd.IEEEAddress)

	ret := mqtt.DeviceDescriptionMessage{
		IEEEAddress: devCmd.IEEEAddress,
		Endpoints:   make([]mqtt.EndpointDescription, 0),
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	descriptor, err := mh.zstack.QueryNodeDescription(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress))
	if err != nil {
		mh.logger.Log("Failed to get node descriptor: %v\n", err)
		return
	}

	ret.LogicalType = uint8(descriptor.LogicalType)
	ret.ManufacturerCode = uint16(descriptor.ManufacturerCode)

	endpoints, err := mh.zstack.QueryNodeEndpoints(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress))
	if err != nil {
		mh.logger.Log("Failed to get node endpoints: %v\n", err)
		return
	}

	for _, endpoint := range endpoints {
		endpointDes, err := mh.zstack.QueryNodeEndpointDescription(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress), endpoint)

		if err != nil {
			mh.logger.Log("Failed to get node endpoint description: %v / %d\n", err, endpoint)
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

	mh.onDeviceDescriptionMessage(ret)
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
		mh.logger.Log("[ProccessGetMessageToDevice] Error Marshal zcl message: %v\n", err)
		return
	}

	err = mh.zstack.SendApplicationMessageToNode(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, false)
	if err != nil {
		mh.logger.Log("[ProccessGetMessageToDevice] Error sending message: %v\n", err)
		return
	}

	mh.logger.Log("[ProccessMessageToDevice] Message (ClusterID: %v, Command: %v) is sent to %v device\n", message.ClusterID, message.CommandIdentifier, devCmd.IEEEAddress)
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
		mh.logger.Log("[ProccessMessageToDevice] Error Local command for ClusterID: %v, Manufacturer: %v, Direction: %v, CommandIdentifier: %v. Error: %v \n",
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
		mh.logger.Log("[ProccessMessageToDevice] Error Marshal zcl message: %v\n", err)
		return
	}

	// timeoutCtx, timeoutCancel := context.WithTimeout(ctx, time.Minute)
	// defer timeoutCancel()

	//err = mh.zstack.SendApplicationMessageToNode(timeoutCtx, zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, true)
	err = mh.zstack.SendApplicationMessageToNode(ctx, zigbee.IEEEAddress(devCmd.IEEEAddress), appMsg, false)
	if err != nil {
		mh.logger.Log("[ProccessMessageToDevice] Error sending message: %v\n", err)
		return
	}

	mh.logger.Log("[ProccessMessageToDevice] Message (ClusterID: %v, Command: %v) is sent to %v device\n", message.ClusterID, message.CommandIdentifier, devCmd.IEEEAddress)
}

func saveNodeDB(znode zigbee.Node, dbObj db.DeviceDB) {
	newDevice := db.Device{
		IEEEAddress:    uint64(znode.IEEEAddress),
		NetworkAddress: uint16(znode.NetworkAddress),
		LogicalType:    uint8(znode.LogicalType),
		LQI:            znode.LQI,
		Depth:          znode.Depth,
		LastDiscovered: znode.LastDiscovered,
		LastReceived:   znode.LastReceived,
	}

	dbObj.SaveDevice(context.Background(), newDevice)
}

func (mh *zigbeeRouter) processNodeJoin(e zigbee.NodeJoinEvent) {
	go saveNodeDB(e.Node, mh.database)

	if mh.onDeviceJoin != nil {
		mh.onDeviceJoin(e)
	}
}

func (mh *zigbeeRouter) processNodeLeave(e zigbee.NodeLeaveEvent) {

	if mh.onDeviceLeave != nil {
		mh.onDeviceLeave(e)
	}
}

func (mh *zigbeeRouter) processNodeUpdate(e zigbee.NodeUpdateEvent) {
	go saveNodeDB(e.Node, mh.database)

	if mh.onDeviceUpdate != nil {
		mh.onDeviceUpdate(e)
	}
}

func (mh *zigbeeRouter) processIncomingMessage(e zigbee.NodeIncomingMessageEvent) {
	go saveNodeDB(e.Node, mh.database)
	msg := e.IncomingMessage
	message, err := mh.zclCommandRegistry.Unmarshal(msg.ApplicationMessage)
	if err != nil {
		mh.logger.Log("[ProcessIncomingMessage] Error parse incomming message: %v\n", err)
		return
	}

	mh.logger.Log("[ProcessIncomingMessage] Incomming command of type (%T) is received. ClusterId=%v, SourceEndpoint=%v\n",
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
	database db.DeviceDB,
	cfg *configuration.Configuration) ZigbeeRouter {
	ret := zigbeeRouter{
		zstack:             z,
		configuration:      cfg,
		zclCommandRegistry: zclCommandRegistry,
		zclDefService:      zclDefService,
		database:           database,
		logger:             logger.GetLogger("[Zigbee Router]"),
	}

	return &ret
}

func (mh *zigbeeRouter) StartAsync(ctx context.Context) {
	go mh.startEventLoop(ctx)
}

func (mh *zigbeeRouter) startEventLoop(ctx context.Context) {
	mh.logger.Log("[Event loop] Start event")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		event, err := mh.zstack.ReadEvent(ctx)
		if err != nil {
			mh.logger.Log("[Event loop] Error read event: %v\n", err)
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			mh.logger.Log("[Event loop] Node join: %v\n", e)
			go mh.processNodeJoin(e)
		case zigbee.NodeLeaveEvent:
			mh.logger.Log("[Event loop] Node leave: %v\n", e)
			go mh.processNodeLeave(e)
		case zigbee.NodeUpdateEvent:
			mh.logger.Log("[Event loop] Node update: %v\n", e)
			go mh.processNodeUpdate(e)
		case zigbee.NodeIncomingMessageEvent:
			mh.logger.Log("[Event loop] Node message: %v\n", e)
			go mh.processIncomingMessage(e)
		}
	}
}
