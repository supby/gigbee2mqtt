package mqtt

import (
	"fmt"
	"log"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/supby/gigbee2mqtt/configuration"
	"github.com/supby/gigbee2mqtt/logger"
)

func NewClient(config *configuration.Configuration) (MqttClient, func()) {
	retClient := defaultMqttClient{
		configuration: config,
		logger:        logger.GetLogger("[MQTT Client]"),
	}

	opts := mqttlib.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MqttConfiguration.Address, config.MqttConfiguration.Port))
	opts.SetClientID(config.MqttConfiguration.RootTopic)
	opts.SetUsername(config.MqttConfiguration.Username)
	opts.SetPassword(config.MqttConfiguration.Password)
	opts.AutoReconnect = true
	opts.SetOrderMatters(false)
	opts.OnConnect = func(client mqttlib.Client) {
		retClient.logger.Log("Connected")
	}
	opts.OnConnectionLost = func(client mqttlib.Client, err error) {
		retClient.logger.Log("Connect lost: %v", err)
	}

	innerClient := mqttlib.NewClient(opts)

	if token := innerClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	token := innerClient.Subscribe(fmt.Sprintf("%v/#", config.MqttConfiguration.RootTopic), 0, retClient.onMessageReceived)
	token.Wait()

	retClient.logger.Log("Connected to MQTT on '%v:%v'", config.MqttConfiguration.Address, config.MqttConfiguration.Port)
	innerClient.Publish(fmt.Sprintf("%v/gateway/status", config.MqttConfiguration.RootTopic), 0, false, "Online")

	retClient.innerClient = innerClient

	return &retClient, func() { retClient.Dispose() }
}

type MqttClient interface {
	Dispose()
	Publish(subTopic string, data []byte)
	Subscribe(callback func(topic string, message []byte))
	UnSubscribe()
}

type defaultMqttClient struct {
	innerClient     mqttlib.Client
	messageCallback func(topic string, message []byte)
	configuration   *configuration.Configuration
	logger          logger.Logger
}

func (cl *defaultMqttClient) Dispose() {
	cl.innerClient.Disconnect(0)
}

func (cl *defaultMqttClient) Publish(subTopic string, data []byte) {
	cl.innerClient.Publish(fmt.Sprintf("%v/%v", cl.configuration.MqttConfiguration.RootTopic, subTopic), 0, false, data)
}

func (cl *defaultMqttClient) Subscribe(callback func(topic string, message []byte)) {
	cl.messageCallback = callback
}

func (cl *defaultMqttClient) UnSubscribe() {
	cl.messageCallback = nil
}

func (cl *defaultMqttClient) onMessageReceived(client mqttlib.Client, msg mqttlib.Message) {
	topic := msg.Topic()
	message := msg.Payload()

	if topic == fmt.Sprintf("%v/gateway/status", cl.configuration.MqttConfiguration.RootTopic) {
		return
	}

	if cl.messageCallback != nil {
		go cl.messageCallback(topic, message)
	}
}
