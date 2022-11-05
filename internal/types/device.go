package types

type DeviceCommandMessage struct {
	IEEEAddress       uint64
	ClusterID         uint16
	Endpoint          uint8
	CommandIdentifier uint8
	CommandData       map[string]interface{}
}

type DeviceGetMessage struct {
	IEEEAddress uint64
	ClusterID   uint16
	Endpoint    uint8
	Attributes  []uint16
}

type DeviceSetMessage struct {
	IEEEAddress uint64
	ClusterID   uint16
	Endpoint    uint8
	Attributes  []AttributeRecord
}

type AttributeRecord struct {
	Id    uint16
	Type  string
	Value interface{}
}

type DeviceExploreMessage struct {
	IEEEAddress uint64
}

type DeviceConfigSetMessage struct {
	PermitJoin bool
}
