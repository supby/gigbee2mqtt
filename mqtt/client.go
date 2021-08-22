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

func InitMQTT(config *configuration.Configuration) (mqttlib.Client, func()) {
	opts := mqttlib.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MqttConfiguration.Address, config.MqttConfiguration.Port))
	opts.SetClientID("gigbee2mqtt")
	opts.SetUsername(config.MqttConfiguration.Username)
	opts.SetPassword(config.MqttConfiguration.Password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqttlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	return client, func() { client.Disconnect(0) }
}
