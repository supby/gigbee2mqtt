package main

import (
	"context"
	"log"
	"time"

	"github.com/shimmeringbee/zigbee"
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

	/* Construct node table, cache of network nodes. */
	t := zstack.NewNodeTable()
	nodes := []zigbee.Node{
		{
			IEEEAddress:    0x00124b000724ae04,
			NetworkAddress: 31211,
			LogicalType:    0x02,
			LQI:            0,
			Depth:          0,
			LastDiscovered: time.Now(),
			LastReceived:   time.Now(),
		},
	}
	t.Load(nodes)

	/* Create a new ZStack struct. */
	z := zstack.New(port, t)

	/* Generate random Zigbee network, on default channel (15) */
	//netCfg, _ := zigbee.GenerateNetworkConfiguration()

	netCfg := zigbee.NetworkConfiguration{
		PANID:         6754,
		ExtendedPANID: zigbee.ExtendedPANID(btoi64([]byte{221, 221, 221, 221, 221, 221, 221, 221})),
		NetworkKey:    zigbee.NetworkKey{0x01, 0x03, 0x05, 0x07, 0x09, 0x0B, 0x0D, 0x0F, 0x00, 0x02, 0x04, 0x06, 0x08, 0x0A, 0x0C, 0x0D},
		Channel:       11,
	}

	/* Obtain context for timeout of initialisation. */
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	/* Initialise ZStack and CC253X */
	err = z.Initialise(ctx, netCfg)
	if err != nil {
		log.Printf("Errir init ZStack: %v\n", err)
		return
	}

	err = z.PermitJoin(ctx, true)
	if err != nil {
		log.Printf("Error permit join: %v\n", err)
		return
	}

	log.Println("Start event loop ====")
	for {
		ctx := context.Background()
		event, err := z.ReadEvent(ctx)

		if err != nil {
			return
		}

		switch e := event.(type) {
		case zigbee.NodeJoinEvent:
			log.Printf("join: %v\n", e.Node)
			go exploreDevice(z, e.Node)
		case zigbee.NodeLeaveEvent:
			log.Printf("leave: %v\n", e.Node)
		case zigbee.NodeUpdateEvent:
			log.Printf("update: %v\n", e.Node)
		case zigbee.NodeIncomingMessageEvent:
			log.Printf("message: %v\n", e)
		}
	}
}

func exploreDevice(z *zstack.ZStack, node zigbee.Node) {
	log.Printf("node %v: querying", node.IEEEAddress)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	descriptor, err := z.QueryNodeDescription(ctx, node.IEEEAddress)

	if err != nil {
		log.Printf("failed to get node descriptor: %v", err)
		return
	}

	log.Printf("node %v: descriptor: %+v", node.IEEEAddress, descriptor)

	endpoints, err := z.QueryNodeEndpoints(ctx, node.IEEEAddress)

	if err != nil {
		log.Printf("failed to get node endpoints: %v", err)
		return
	}

	log.Printf("node %v: endpoints: %+v", node.IEEEAddress, endpoints)

	for _, endpoint := range endpoints {
		endpointDes, err := z.QueryNodeEndpointDescription(ctx, node.IEEEAddress, endpoint)

		if err != nil {
			log.Printf("failed to get node endpoint description: %v / %d", err, endpoint)
		} else {
			log.Printf("node %v: endpoint: %d desc: %+v", node.IEEEAddress, endpoint, endpointDes)
		}
	}
}

func btoi64(val []byte) uint64 {
	r := uint64(0)
	for i := uint64(0); i < 8; i++ {
		r |= uint64(val[i]) << (8 * i)
	}
	return r
}
