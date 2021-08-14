package main

import (
	"context"
	"log"
	"time"

	"github.com/shimmeringbee/zigbee"
	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/db"

	//"github.com/supby/gigbee2mqtt/zstack"
	"github.com/shimmeringbee/zstack"

	"go.bug.st/serial.v1"
)

// Example of message:
// MQTT publish: topic 'zigbee2mqtt/0x00124b000724ae04',
// payload '{"cluster":"ssIasZone","data":{"batteryPercentageRemaining":200,"batteryVoltage":30,"zoneStatus":1},"device":{"friendlyName":"0x00124b000724ae04","ieeeAddr":"0x00124b000724ae04","model":"unknown","networkAddress":31211,"type":"Unknown"},"endpoint":{"ID":1,"_binds":[],"_configuredReportings":[],"clusters":{"genPowerCfg":{"attributes":{"batteryPercentageRemaining":200,"batteryVoltage":30}},"ssIasZone":{"attributes":{"zoneStatus":1}}},"deviceIeeeAddress":"0x00124b000724ae04","deviceNetworkAddress":31211,"inputClusters":[],"meta":{},"outputClusters":[]},"groupID":0,"linkquality":52,"meta":{"frameControl":{"direction":1,"disableDefaultResponse":true,"frameType":0,"manufacturerSpecific":false,"reservedBits":0},"manufacturerCode":null,"zclTransactionSequenceNumber":219},"type":"attributeReport"}'

func main() {
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open("/dev/ttyACM0", mode)
	if err != nil {
		log.Fatal(err)
	}
	port.SetRTS(true)

	db1 := db.Init("./db.json")

	/* Construct node table, cache of network nodes. */
	t := zstack.NewNodeTable()
	loadNodeTableFromDB(t, db1)

	/* Create a new ZStack struct. */
	z := zstack.New(port, t)

	/* Generate random Zigbee network, on default channel (15) */
	//netCfg, _ := zigbee.GenerateNetworkConfiguration()
	cfg := configuration.Init("./configuration.yaml")
	netCfg := zigbee.NetworkConfiguration{
		PANID:         zigbee.PANID(cfg.NetworkConfiguration.PANID),
		ExtendedPANID: zigbee.ExtendedPANID(cfg.NetworkConfiguration.ExtendedPANID),
		NetworkKey:    cfg.NetworkConfiguration.NetworkKey,
		Channel:       cfg.NetworkConfiguration.Channel,
	}

	/* Obtain context for timeout of initialisation. */
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	/* Initialise ZStack and CC253X */
	err = z.Initialise(ctx, netCfg)
	if err != nil {
		log.Fatal(err)
	}

	err = z.PermitJoin(ctx, true)
	if err != nil {
		log.Printf("Error permit join: %v\n", err)
	}

	if err := z.RegisterAdapterEndpoint(ctx, 1, zigbee.ProfileHomeAutomation, 1, 1, []zigbee.ClusterID{}, []zigbee.ClusterID{}); err != nil {
		log.Fatal(err)
	}

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
			go saveNodeDB(e.Node, db1)
		case zigbee.NodeLeaveEvent:
			log.Printf("leave: %v\n", e.Node)
		case zigbee.NodeUpdateEvent:
			log.Printf("update: %v\n", e.Node)
			go saveNodeDB(e.Node, db1)
		case zigbee.NodeIncomingMessageEvent:
			log.Printf("message: %v\n", e)
		}
	}
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
