package controlplane

import (
	"fmt"
	"os"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	mqtt_paho "github.com/Laboratory-for-Safe-and-Secure-Systems/paho.mqtt.golang"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
)

type mqtt_client struct {
	client  mqtt_paho.Client
	ep      asl.EndpointConfig
	logger  *zerolog.Logger
	address string
	id      string
}

var messagePubHandler mqtt_paho.MessageHandler = func(client mqtt_paho.Client, msg mqtt_paho.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt_paho.OnConnectHandler = func(client mqtt_paho.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt_paho.ConnectionLostHandler = func(client mqtt_paho.Client, err error) {
	fmt.Printf("Connection lost: %v\n\n\n\n\n\n", err)
}

var reconHandler mqtt_paho.ReconnectHandler = func(client mqtt_paho.Client, opts *mqtt_paho.ClientOptions) {
	fmt.Print("RECON RECON %v\n")
}

func new_mqtt_client(mqtt_cfg types.ControlPlaneConfig) *mqtt_client {
	zerologger := zerolog.New(os.Stdout).Level(zerolog.Level(mqtt_cfg.Log.Level))

	client_opts := mqtt_paho.NewClientOptions()
	client_opts = client_opts.SetCleanSession(true)
	client_opts.AddBroker("tls://" + mqtt_cfg.Address)
	client_opts.SetClientID("controller")
	client_opts.SetDefaultPublishHandler(messagePubHandler)
	client_opts.OnConnect = connectHandler
	client_opts.OnConnectionLost = connectLostHandler
	client_opts.OnReconnecting = reconHandler
	client_opts.CustomOpenConnectionFn = mqtt_paho.Get_custom_function(mqtt_cfg.EndpointConfig)
	client := mqtt_paho.NewClient(client_opts)

	return &mqtt_client{
		client:  client,
		ep:      mqtt_cfg.EndpointConfig,
		logger:  &zerologger,
		address: mqtt_cfg.Address,
	}
}
