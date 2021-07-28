package main

import (
	"context"
	"log"
	"time"

	"github.com/shimmeringbee/zigbee"
	"github.com/shimmeringbee/zstack"
	"go.bug.st/serial.v1"
)

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
