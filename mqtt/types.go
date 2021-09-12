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
	IEEEAddress    uint64
	ClusterCommand ClusterCommandData
}

type ClusterCommandData struct {
	ID                uint16
	Name              string
	Type              string
	CommandIdentifier string
	CommandData       string
}
