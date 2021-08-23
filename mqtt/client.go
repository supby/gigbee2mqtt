package mqtt

import (
	"fmt"
	"log"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/supby/gigbee2mqtt/configuration"
)

var messagePubHandler mqttlib.MessageHandler = func(client mqttlib.Client, msg mqttlib.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqttlib.OnConnectHandler = func(client mqttlib.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqttlib.ConnectionLostHandler = func(client mqttlib.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func NewClient(config *configuration.Configuration) (*Client, func()) {
	opts := mqttlib.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MqttConfiguration.Address, config.MqttConfiguration.Port))
	opts.SetClientID("gigbee2mqtt")
	opts.SetUsername(config.MqttConfiguration.Username)
	opts.SetPassword(config.MqttConfiguration.Password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	innerClient := mqttlib.NewClient(opts)
	if token := innerClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	client := Client{
		innerClient: innerClient,
	}

	return &client, func() { client.Dispose() }
}

type Client struct {
	innerClient mqttlib.Client
}

func (cl *Client) Dispose() {
	cl.innerClient.Disconnect(0)
}

func (cl *Client) Publish(topic string, data []byte) {
	cl.innerClient.Publish(topic, 0, false, data)
}
