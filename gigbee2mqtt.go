package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"

	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"
	"github.com/supby/gigbee2mqtt/handler"
	"github.com/supby/gigbee2mqtt/mqtt"
	"github.com/supby/gigbee2mqtt/zcldef"

	"go.bug.st/serial.v1"
)

// Example of message:
// MQTT publish: topic 'zigbee2mqtt/0x00124b000724ae04',
// payload '{"cluster":"ssIasZone","data":{"batteryPercentageRemaining":200,"batteryVoltage":30,"zoneStatus":1},"device":{"friendlyName":"0x00124b000724ae04","ieeeAddr":"0x00124b000724ae04","model":"unknown","networkAddress":31211,"type":"Unknown"},"endpoint":{"ID":1,"_binds":[],"_configuredReportings":[],"clusters":{"genPowerCfg":{"attributes":{"batteryPercentageRemaining":200,"batteryVoltage":30}},"ssIasZone":{"attributes":{"zoneStatus":1}}},"deviceIeeeAddress":"0x00124b000724ae04","deviceNetworkAddress":31211,"inputClusters":[],"meta":{},"outputClusters":[]},"groupID":0,"linkquality":52,"meta":{"frameControl":{"direction":1,"disableDefaultResponse":true,"frameType":0,"manufacturerSpecific":false,"reservedBits":0},"manufacturerCode":null,"zclTransactionSequenceNumber":219},"type":"attributeReport"}'

func main() {
	cfg := configuration.Init("./configuration.yaml")

	db1 := db.Init("./db.json")

	z := initZStack(cfg, db1)

	zclCommandRegistry := zcl.NewCommandRegistry()
	global.Register(zclCommandRegistry)

	zclDefMap := zcldef.Load("./zcldef/zcldef.json")
	if zclDefMap == nil {
		log.Fatal("Error loading ZCL map")
	}

	mqttClient, mqttDisconnect := mqtt.NewClient(cfg, mqqtMessage)
	defer mqttDisconnect()

	messageHsandler := handler.Create(zclCommandRegistry, zclDefMap, mqttClient, db1, cfg)

	startEventLoop(z, messageHsandler)
}

func mqqtMessage(topic string, message []byte) {

	var devMsg mqtt.DeviceMessage
	json.Unmarshal(message, &devMsg)

	ieeeAddress := strings.Split(topic, "/")[0]
}

func startEventLoop(z *zstack.ZStack, messageHandler *handler.MessageHandler) {
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
			go messageHandler.ProcessNodeJoin(e)
		case zigbee.NodeLeaveEvent:
			log.Printf("leave: %v\n", e.Node)
		case zigbee.NodeUpdateEvent:
			log.Printf("update: %v\n", e.Node)
			go messageHandler.ProcessNodeUpdate(e)
		case zigbee.NodeIncomingMessageEvent:
			log.Printf("message: %v\n", e)
			go messageHandler.ProcessIncomingMessage(e)
		}
	}
}

func initZStack(cfg *configuration.Configuration, db1 *db.DB) *zstack.ZStack {
	mode := &serial.Mode{
		BaudRate: int(cfg.SerialConfiguration.BaudRate),
	}

	port, err := serial.Open(cfg.SerialConfiguration.PortName, mode)
	if err != nil {
		log.Fatal(err)
	}
	port.SetRTS(true)

	/* Construct node table, cache of network nodes. */
	t := zstack.NewNodeTable()
	loadNodeTableFromDB(t, db1)

	/* Create a new ZStack struct. */
	z := zstack.New(port, t)

	netCfg := zigbee.NetworkConfiguration{
		PANID:         zigbee.PANID(cfg.ZNetworkConfiguration.PANID),
		ExtendedPANID: zigbee.ExtendedPANID(cfg.ZNetworkConfiguration.ExtendedPANID),
		NetworkKey:    cfg.ZNetworkConfiguration.NetworkKey,
		Channel:       cfg.ZNetworkConfiguration.Channel,
	}

	/* Obtain context for timeout of initialisation. */
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	/* Initialise ZStack and CC253X */
	err = z.Initialise(ctx, netCfg)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.PermitJoin {
		err = z.PermitJoin(ctx, true)
		if err != nil {
			log.Printf("Error permit join: %v\n", err)
		}
	}

	if err := z.RegisterAdapterEndpoint(ctx, 1, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		log.Fatal(err)
	}

	return z
}

func loadNodeTableFromDB(t *zstack.NodeTable, dbObj *db.DB) {
	znodes := make([]zigbee.Node, len(dbObj.Nodes))

	for i, dbNode := range dbObj.Nodes {
		znodes[i] = zigbee.Node{
			IEEEAddress:    zigbee.IEEEAddress(dbNode.IEEEAddress),
			NetworkAddress: zigbee.NetworkAddress(dbNode.NetworkAddress),
			LogicalType:    zigbee.LogicalType(dbNode.LogicalType),
			LQI:            dbNode.LQI,
			Depth:          dbNode.Depth,
			LastDiscovered: dbNode.LastDiscovered,
			LastReceived:   dbNode.LastReceived,
		}
	}

	t.Load(znodes)
}
