package db

import "time"

type Node struct {
	IEEEAddress    uint64
	NetworkAddress uint16
	LogicalType    uint8
	LQI            uint8
	Depth          uint8
	LastDiscovered time.Time
	LastReceived   time.Time
}
