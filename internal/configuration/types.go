package configuration

type ZNetworkConfiguration struct {
	PANID         uint16
	ExtendedPANID uint64
	NetworkKey    [16]byte
	Channel       uint8
}

type MqttConfiguration struct {
	Address   string
	Port      uint16
	RootTopic string
	Username  string
	Password  string
}

type SerialConfiguration struct {
	PortName string
	BaudRate uint32
}

type Configuration struct {
	ZNetworkConfiguration ZNetworkConfiguration
	MqttConfiguration     MqttConfiguration
	SerialConfiguration   SerialConfiguration
	PermitJoin            bool
	LogLevel              int // info=0, warn=1, error=2, debug=3
}
