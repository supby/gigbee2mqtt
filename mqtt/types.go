package mqtt

type DeviceAttributesReportMessage struct {
	IEEEAddress       uint64
	LinkQuality       uint8
	ClusterAttributes ClusterAttributesData
}

type ClusterAttributesData struct {
	ID         uint16
	Name       string
	Type       string
	Attributes map[string]interface{}
}
type DeviceSetMessage struct {
	ClusterID         uint16
	Endpoint          uint8
	CommandIdentifier uint8
	CommandData       interface{}
}
