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

	port, err := serial.Open("/dev/ttyUSB0", mode)
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
	netCfg, _ := zigbee.GenerateNetworkConfiguration()

	/* Obtain context for timeout of initialisation. */
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	/* Initialise ZStack and CC253X */
	err = z.Initialise(ctx, netCfg)
}
