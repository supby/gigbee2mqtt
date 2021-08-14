package utils

import (
	"context"
	"log"
	"time"

	"github.com/shimmeringbee/zigbee"
	//"github.com/supby/gigbee2mqtt/zstack"
	"github.com/shimmeringbee/zstack"
)

func Btoi64(val []byte) uint64 {
	r := uint64(0)
	for i := uint64(0); i < 8; i++ {
		r |= uint64(val[i]) << (8 * i)
	}
	return r
}

func ExploreDevice(z *zstack.ZStack, node zigbee.Node) {
	log.Printf("node %v: querying", node.IEEEAddress)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
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

	log.Printf("Exploring of %v finished", node.IEEEAddress)
}
