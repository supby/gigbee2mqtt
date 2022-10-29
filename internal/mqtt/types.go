package mqtt

type DeviceAttributesReportMessage struct {
	ClusterID         uint16
	ClusterName       string
	ClusterType       string
	ClusterAttributes interface{}
}

type DeviceSetMessage struct {
	ClusterID         uint16
	Endpoint          uint8
	CommandIdentifier uint8
	CommandData       map[string]interface{}
}

type DeviceGetMessage struct {
	ClusterID  uint16
	Endpoint   uint8
	Attributes []uint16
}

type DeviceDefaultResponseMessage struct {
	ClusterID         uint16
	CommandIdentifier uint8
	Status            uint8
}

type DeviceMessage struct {
	IEEEAddress uint64
	LinkQuality uint8
	Message     interface{}
}

type DeviceDescriptionMessage struct {
	IEEEAddress      uint64
	LogicalType      uint8
	ManufacturerCode uint16
	Endpoints        []EndpointDescription
}

type EndpointDescription struct {
	Endpoint       uint8
	ProfileID      uint16
	DeviceID       uint16
	DeviceVersion  uint8
	InClusterList  []uint16
	OutClusterList []uint16
}

type SetGatewayConfig struct {
	PermitJoin bool
}
