package mqtt

type DeviceAttributesReportMessage struct {
	ClusterID         uint16
	ClusterName       string
	ClusterType       string
	ClusterAttributes map[string]interface{}
}

type DeviceSetMessage struct {
	ClusterID         uint16
	Endpoint          uint8
	CommandIdentifier uint8
	CommandData       map[string]interface{}
}

type DeviceGetMessage struct {
	ClusterID uint16
	Endpoint  uint8
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

type SetGatewayConfig struct {
	PermitJoin bool
}
