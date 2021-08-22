package configuration

import (
	"github.com/supby/gigbee2mqtt/utils"
)

type ZNetworkConfiguration struct {
	PANID         uint16
	ExtendedPANID uint64
	NetworkKey    [16]byte
	Channel       uint8
}

type MqttConfiguration struct {
	Address  string
	Port     uint16
	Topic    string
	Username string
	Password string
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
}

func Init(filename string) *Configuration {

	cfg := Configuration{
		ZNetworkConfiguration: ZNetworkConfiguration{
			PANID:         9945,
			ExtendedPANID: utils.Btoi64([]byte{125, 221, 221, 125, 221, 221, 125, 221}),
			NetworkKey:    [16]byte{0x01, 0x03, 0x05, 0x07, 0x09, 0x0B, 0x0D, 0x0F, 0x00, 0x02, 0x04, 0x06, 0x08, 0x0A, 0x0C, 0x0D},
			Channel:       15,
		},
		SerialConfiguration: SerialConfiguration{
			PortName: "/dev/ttyACM0",
			BaudRate: 115200,
		},
		PermitJoin: true,
		MqttConfiguration: MqttConfiguration{
			Address: "192.168.1.25",
			Port:    1883,
			Topic:   "gigbee2mqtt",
		},
	}

	return &cfg
}
