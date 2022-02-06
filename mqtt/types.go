package mqtt

type DeviceAttributesReportMessage struct {
	IEEEAddress       uint64
	LinkQuality       uint8
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

type DeviceDefaultResponseMessage struct {
	IEEEAddress       uint64
	LinkQuality       uint8
	ClusterID         uint16
	CommandIdentifier uint8
	Status            uint8
}
