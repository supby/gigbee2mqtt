package configuration

import (
	"github.com/supby/gigbee2mqtt/utils"
)

type NetworkConfiguration struct {
	PANID         uint16
	ExtendedPANID uint64
	NetworkKey    [16]byte
	Channel       uint8
}

type Configuration struct {
	NetworkConfiguration NetworkConfiguration
}

func Init(filename string) *Configuration {

	cfg := Configuration{
		NetworkConfiguration: NetworkConfiguration{
			PANID:         9945,
			ExtendedPANID: utils.Btoi64([]byte{125, 221, 221, 125, 221, 221, 125, 221}),
			NetworkKey:    [16]byte{0x01, 0x03, 0x05, 0x07, 0x09, 0x0B, 0x0D, 0x0F, 0x00, 0x02, 0x04, 0x06, 0x08, 0x0A, 0x0C, 0x0D},
			Channel:       15,
		},
	}

	return &cfg
}
