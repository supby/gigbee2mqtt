package configuration

import (
	"fmt"
	"os"

	"github.com/supby/gigbee2mqtt/utils"
	"gopkg.in/yaml.v2"
)

type configurationService struct {
	filename      string
	configuration Configuration
}

func (cs *configurationService) GetConfiguration() Configuration {
	return cs.configuration
}

func (cs *configurationService) Update(updatedConfig Configuration) error {
	return nil
}

func Init(filename string) (ConfigurationService, error) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Configuration file '%v' does not exist", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := Configuration{
		ZNetworkConfiguration: ZNetworkConfiguration{
			PANID:         9945,
			ExtendedPANID: utils.Btoi64([]byte{125, 221, 221, 125, 221, 221, 125, 221}),
			NetworkKey:    [16]byte{0x01, 0x03, 0x05, 0x07, 0x09, 0x0B, 0x0D, 0x0F, 0x00, 0x02, 0x04, 0x06, 0x08, 0x0A, 0x0C, 0x0D},
			Channel:       15,
		},
		SerialConfiguration: SerialConfiguration{
			BaudRate: 115200,
		},
		PermitJoin: true,
		MqttConfiguration: MqttConfiguration{
			Port:      1883,
			RootTopic: "gigbee2mqtt",
		},
	}

	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return nil, err
	}

	return &configurationService{
		filename:      filename,
		configuration: cfg,
	}, nil
}
