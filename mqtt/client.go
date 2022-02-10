package mqtt

import (
	"fmt"
	"log"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/supby/gigbee2mqtt/configuration"
)

var connectHandler mqttlib.OnConnectHandler = func(client mqttlib.Client) {
	log.Println("Connected")
}

var connectLostHandler mqttlib.ConnectionLostHandler = func(client mqttlib.Client, err error) {
	log.Printf("Connect lost: %v", err)
}

func NewClient(config *configuration.Configuration) (MqttClient, func()) {
	retClient := defaultMqttClient{
		configuration: config,
	}

	opts := mqttlib.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MqttConfiguration.Address, config.MqttConfiguration.Port))
	opts.SetClientID("gigbee2mqtt")
	opts.SetUsername(config.MqttConfiguration.Username)
	opts.SetPassword(config.MqttConfiguration.Password)
	opts.AutoReconnect = true
	opts.SetOrderMatters(false)
	opts.SetDefaultPublishHandler(func(client mqttlib.Client, msg mqttlib.Message) {
		log.Printf("[MQTT Client] Received message: %s for topic: %s\n", msg.Payload(), msg.Topic())
		retClient.onMessageReceived(msg.Topic(), msg.Payload())
	})
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	innerClient := mqttlib.NewClient(opts)
	if token := innerClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	token := innerClient.Subscribe(fmt.Sprintf("%v/#", config.MqttConfiguration.Topic), 0, nil)
	token.Wait()

	log.Printf("[MQTT Client] Connected to MQTT on '%v:%v'", config.MqttConfiguration.Address, config.MqttConfiguration.Port)
	innerClient.Publish(fmt.Sprintf("%v/gateway", config.MqttConfiguration.Topic), 0, false, "Online")

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
}

func (cl *defaultMqttClient) Dispose() {
	cl.innerClient.Disconnect(0)
}

func (cl *defaultMqttClient) Publish(subTopic string, data []byte) {
	cl.innerClient.Publish(fmt.Sprintf("%v/%v", cl.configuration.MqttConfiguration.Topic, subTopic), 0, false, data)
}

func (cl *defaultMqttClient) Subscribe(callback func(topic string, message []byte)) {
	cl.messageCallback = callback
}

func (cl *defaultMqttClient) UnSubscribe() {
	cl.messageCallback = nil
}

func (cl *defaultMqttClient) onMessageReceived(topic string, message []byte) {
	if topic == fmt.Sprintf("%v/gateway", cl.configuration.MqttConfiguration.Topic) {
		return
	}

	if cl.messageCallback != nil {
		go cl.messageCallback(topic, message)
	}
}
