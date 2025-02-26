package control_plane

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	mqtt_paho "github.com/Laboratory-for-Safe-and-Secure-Systems/paho.mqtt.golang"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mqtt_client struct {
	client  mqtt_paho.Client
	ep      asl.EndpointConfig
	logger  *zerolog.Logger
	address string
	id      string
	v1.UnimplementedControlPlaneServer
}

/********************************** Handler *******************************************/

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

/**********************************End  Handler *******************************************/

func new_mqtt_client(mqtt_cfg types.ControlPlaneConfig) *mqtt_client {
	zerologger := zerolog.New(os.Stdout).Level(zerolog.Level(mqtt_cfg.Log.Level))

	client_opts := mqtt_paho.NewClientOptions()
	//clean session true means that the client will not receive any messages saved messages from the broker
	client_opts = client_opts.SetCleanSession(true)
	client_opts.AddBroker("tls://" + mqtt_cfg.Address)
	client_opts.SetClientID("controller")
	client_opts.SetDefaultPublishHandler(messagePubHandler)
	client_opts.OnConnect = connectHandler
	client_opts.OnConnectionLost = connectLostHandler
	client_opts.OnReconnecting = reconHandler
	client_opts.CustomOpenConnectionFn = mqtt_paho.Get_custom_function(mqtt_cfg.EndpointConfig)
	client_opts.SetProtocolVersion(3)
	//qos
	client := mqtt_paho.NewClient(client_opts)

	return &mqtt_client{
		client:  client,
		ep:      mqtt_cfg.EndpointConfig,
		logger:  &zerologger,
		address: mqtt_cfg.Address,
	}
}

func (c *mqtt_client) serve() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	sub_token := c.client.Subscribe("test", 0, nil)
	c.client.SubscribeMultiple()

	return nil
}

/********************************** grpc service for control_plane *******************************************/
func (c *mqtt_client) UpdateNode(req *v1.NodeUpdate, stream grpc.ServerStreamingServer[v1.UpdateResponse]) error {
	// Create channel for the stream and done signal
	streamChan := make(chan v1.UpdateState)
	doneChan := make(chan error)
	timeout := time.After(40 * time.Second) // Add a reasonable timeout

	serialNumber := req.NodeUpdateItem.SerialNumber
	// Use high qos
	topicQos := serialNumber + "/control/qos"
	topicConfig := serialNumber + "/config" // Fixed the topic path

	// Subscribe to the topic
	qosToken := c.client.Subscribe(topicQos, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		c.logger.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		payload := msg.Payload()
		// Parse payload to v1.UpdateState
		var updateState v1.UpdateState
		err := json.Unmarshal(payload, &updateState)
		if err != nil {
			c.logger.Err(err).Msg("error unmarshalling update state")
			doneChan <- err
			return
		}
		streamChan <- updateState
	})

	if err := qosToken.Error(); err != nil {
		return err
	}
	qosToken.Wait()

	jsonReq, err := json.Marshal(req)
	if err != nil {
		c.logger.Err(err).Msg("error marshalling request")
		return err
	}

	// Publish config to client
	configToken := c.client.Publish(topicConfig, 2, false, jsonReq)
	configToken.Wait()
	if configToken.Error() != nil {
		c.logger.Err(configToken.Error()).Msg("error publishing update to node")
		return configToken.Error()
	}

	go func() {
		// Send initial PUBLISHED state
		err := stream.Send(&v1.UpdateResponse{
			UpdateState: v1.UpdateState_PUBLISHED,
			Timestamp:   timestamppb.New(time.Now()),
			TxId:        req.Transaction.TxId,
		})
		if err != nil {
			doneChan <- err
			return
		}

		// Process state updates
		for {
			select {
			case updateState := <-streamChan:
				// Send update to gRPC stream
				err := stream.Send(&v1.UpdateResponse{
					UpdateState: updateState,
					Timestamp:   timestamppb.New(time.Now()),
					TxId:        req.Transaction.TxId,
				})
				if err != nil {
					doneChan <- err
					return
				}

				c.logger.Debug().Msgf("Update to node %s state: %s; tx_id: %s",
					serialNumber, updateState.String(), req.Transaction.TxId)

				// Terminal states
				if updateState == v1.UpdateState_NODE_APPLIED ||
					updateState == v1.UpdateState_ERROR {
					c.client.Unsubscribe(topicQos)
					doneChan <- nil
					return
				}
			}
		}
	}()

	// Wait for completion or timeout
	select {
	//nil in case of success
	case err := <-doneChan:
		return err
	case <-timeout:
		c.client.Unsubscribe(topicQos)
		return fmt.Errorf("update operation timed out for node %s", serialNumber)
	}
}

func (c *mqtt_client) UpdateFleet(req *v1.FleetUpdate, stream grpc.ServerStreamingServer[v1.UpdateResponse]) error {
	// Create channel for the stream and done signal
	streamChan := make(chan v1.UpdateState)
	doneChan := make(chan error)
	timeout := time.After(40 * time.Second) // Add a reasonable timeout

	// Use high qos
	topicQos := "+/control/sync/qos"

	type control_message struct {
		serial_number string
		update_state  v1.UpdateState
		tx_id         string
		timestamp     time.Time
	}
	type control struct {
		mutex   sync.Mutex
		channel chan control_message
	}
	ctrl := control{
		mutex:   sync.Mutex{},
		channel: make(chan control_message, 4),
	}

	qosToken := c.client.Subscribe(topicQos, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		var updateState v1.UpdateState
		c.logger.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		// get serial number
		// get first path
		paths := strings.Split(msg.Topic(), "/")
		serialNumber := paths[0]
		err := json.Unmarshal(msg.Payload(), &updateState)
		if err != nil {
			c.logger.Err(err).Msg("error unmarshalling update state")
			doneChan <- err
			return
		}
		control_message := control_message{
			serial_number: serialNumber,
			update_state:  updateState,
			timestamp:     time.Now(),
		}

		ctrl.mutex.Lock()
		ctrl.channel <- control_message
		ctrl.mutex.Unlock()
		if err != nil {
			c.logger.Err(err).Msg("error unmarshalling update state")
			doneChan <- err
			return
		}
		streamChan <- updateState
	})

	if err := qosToken.Error(); err != nil {
		return err
	}
	qosToken.Wait()

	jsonReq, err := json.Marshal(req)
	if err != nil {
		c.logger.Err(err).Msg("error marshalling request")
		return err
	}

	go func() {
		// Send initial PUBLISHED state
		err := stream.Send(&v1.UpdateResponse{
			UpdateState: v1.UpdateState_PUBLISHED,
			Timestamp:   timestamppb.New(time.Now()),
			TxId:        req.Transaction.TxId,
		})
		if err != nil {
			doneChan <- err
			return
		}
		type control_message_handler struct {
			number_nodes int
			nodes        map[string]control_message
			global_state v1.UpdateState
		}
		handler := control_message_handler{
			number_nodes: len(req.NodeUpdateItems),
			nodes:        make(map[string]control_message),
			global_state: v1.UpdateState_PUBLISHED,
		}

		// Process state updates
		for {
			select {
			case control_message := <-ctrl.channel:
				switch control_message.update_state {
				case v1.UpdateState_PUBLISHED:
					c.logger.Debug().Msgf("Send config to node: %s", control_message.serial_number)
					handler.nodes[control_message.serial_number] = control_message
				case v1.UpdateState_NODE_RECEIVED:
					handler.nodes[control_message.serial_number] = control_message
					stream.Send(&v1.UpdateResponse{
						UpdateState: v1.UpdateState_NODE_RECEIVED,
						Timestamp:   timestamppb.New(time.Now()),
						TxId:        req.Transaction.TxId,
					})
				case v1.UpdateState_UPDATE_APPLICABLE:
					handler.nodes[control_message.serial_number] = control_message
					for _, node := range req.NodeUpdateItems {
						if _, ok := handler.nodes[node.SerialNumber]; !ok {
							continue
						}
					}
					stream.Send(&v1.UpdateResponse{
						UpdateState: v1.UpdateState_UPDATE_APPLICABLE,
						Timestamp:   timestamppb.New(time.Now()),
						TxId:        req.Transaction.TxId,
					})
					c.logger.Info().Msgf("Nodes are complient with update. Server sends apply request")
					for _, node := range req.NodeUpdateItems {
						topicApply := node.SerialNumber + "/sync/config"
						token := c.client.Publish(topicApply, 2, false, []byte("true"))
						token.Wait()
						if token.Error() != nil {
							c.logger.Err(token.Error()).Msg("error publishing apply request to node")
							doneChan <- token.Error()
							return
						}
					}
				case v1.UpdateState_NODE_APPLIED:
					handler.nodes[control_message.serial_number] = control_message
					for _, node := range req.NodeUpdateItems {
						if _, ok := handler.nodes[node.SerialNumber]; !ok {
							continue
						}
					}
					stream.Send(&v1.UpdateResponse{
						UpdateState: v1.UpdateState_NODE_APPLIED,
						Timestamp:   timestamppb.New(time.Now()),
						TxId:        req.Transaction.TxId,
					})
					c.logger.Info().Msgf("All nodes applied update")
					//done
					doneChan <- nil
					return

				case v1.UpdateState_ERROR:
					handler.nodes[control_message.serial_number] = control_message

					for _, node := range req.NodeUpdateItems {
						//send rollback
						topicRollback := node.SerialNumber + "/sync/config"
						token := c.client.Publish(topicRollback, 2, false, []byte("false"))
						token.Wait()
						if token.Error() != nil {
							c.logger.Err(token.Error()).Msg("error publishing rollback request to node")
							doneChan <- token.Error()
							return
						}

						stream.Send(&v1.UpdateResponse{
							UpdateState: v1.UpdateState_ERROR,
							Timestamp:   timestamppb.New(time.Now()),
							TxId:        req.Transaction.TxId,
						})
						c.logger.Info().Msgf("All nodes applied update")
						//done
						doneChan <- nil
						return

					}

				}
			}
		}
	}()

	// Publish config to client
	for _, node := range req.NodeUpdateItems {
		topicConfig := node.SerialNumber + "/sync/config"
		configToken := c.client.Publish(topicConfig, 2, false, jsonReq)

		if configToken.Error() != nil {
			c.logger.Err(configToken.Error()).Msg("error publishing update to node")
			return configToken.Error()
		}
		configToken.Wait()
		ctrl.channel <- control_message{
			serial_number: node.SerialNumber,
			update_state:  v1.UpdateState_PUBLISHED,
			tx_id:         req.Transaction.TxId,
			timestamp:     time.Now(),
		}
	}

	// Wait for completion or timeout
	select {
	//nil in case of success
	case err := <-doneChan:
		return err
	case <-timeout:
		c.client.Unsubscribe(topicQos)
		return fmt.Errorf("update operation timed out for node %s", serialNumber)
	}
}
