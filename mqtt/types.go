package mqtt

type DeviceMessage struct {
	IEEEAddress uint64
	LinkQuality uint8
	Cluster     ClusterData
}

type ClusterData struct {
	ID         uint16
	Name       string
	Type       string
	Attributes map[string]interface{}
}
