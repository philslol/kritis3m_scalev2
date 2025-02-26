package control_plane

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	mqtt_paho "github.com/Laboratory-for-Safe-and-Secure-Systems/paho.mqtt.golang"
	"github.com/golang/protobuf/ptypes/empty"
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
type client struct {
	name   string
	client *mqtt_paho.Client
}

type mqtt_factory struct {
	mu            sync.Mutex
	client_config mqtt_paho.ClientOptions
	ep            asl.EndpointConfig
	clients       []*mqtt_paho.Client
}

func (f *mqtt_factory) GetClient(id string) *mqtt_client {
	f.mu.Lock()
	defer f.mu.Unlock()
	return nil
}
func (f *mqtt_factory) NewClient(id string) *mqtt_client {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clients[id] = new_mqtt_client(id)
	return f.clients[id]
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

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		zerologger.Err(token.Error()).Msg("error connecting to mqtt broker")
		return nil
	}

	return &mqtt_client{
		client:  client,
		ep:      mqtt_cfg.EndpointConfig,
		logger:  &zerologger,
		address: mqtt_cfg.Address,
	}
}

/********************************** grpc service for control_plane *******************************************/
func (c *mqtt_client) UpdateFleet(req *v1.FleetUpdate, stream grpc.ServerStreamingServer[v1.UpdateResponse]) error {
	c.logger.Info().Msgf("Starting fleet update for %d nodes, tx_id: %s", len(req.NodeUpdateItems), req.Transaction.TxId)

	// Create communication channels
	nodeChan := make(chan nodeUpdateMessage, len(req.NodeUpdateItems)*4) // Buffer for multiple messages per node
	doneChan := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // More reasonable timeout for fleet updates
	defer cancel()

	// Initialize fleet status tracker
	status := &fleetStatus{
		nodes:           make(map[string]v1.UpdateState),
		expectedNodes:   make(map[string]bool),
		lastGlobalState: v1.UpdateState_UNKNOWN,
	}

	// Register expected nodes
	for _, node := range req.NodeUpdateItems {
		status.expectedNodes[node.SerialNumber] = true
	}

	// Topic for status updates from nodes
	topicQos := "+/control/sync/qos"

	// Subscribe to all node responses
	qosToken := c.client.Subscribe(topicQos, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		// Extract serial number from topic
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) < 4 {
			c.logger.Error().Msgf("Invalid topic format: %s", msg.Topic())
			return
		}
		serialNumber := parts[0]

		// Parse update state
		var updateState v1.UpdateState
		var errorMsg string

		// Try to unmarshal payload
		err := json.Unmarshal(msg.Payload(), &updateState)
		if err != nil {
			// Try to parse as error message
			var errorPayload struct {
				State string `json:"state"`
				Error string `json:"error"`
			}

			if err := json.Unmarshal(msg.Payload(), &errorPayload); err == nil {
				if errorPayload.State == "ERROR" {
					updateState = v1.UpdateState_ERROR
					errorMsg = errorPayload.Error
				}
			} else {
				c.logger.Error().Err(err).Str("payload", string(msg.Payload())).Msg("Failed to parse node response")
				return
			}
		}

		c.logger.Debug().
			Str("node", serialNumber).
			Str("state", updateState.String()).
			Msg("Node update status received")

		// Send to processing channel
		select {
		case nodeChan <- nodeUpdateMessage{
			SerialNumber: serialNumber,
			State:        updateState,
			TxId:         req.Transaction.TxId,
			Timestamp:    time.Now(),
			Error:        errorMsg,
		}:
		case <-ctx.Done():
			return
		}
	})

	if err := qosToken.Error(); err != nil {
		return fmt.Errorf("failed to subscribe to node responses: %w", err)
	}
	qosToken.Wait()

	// Prepare update payload
	jsonReq, err := json.Marshal(req)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to marshal update request")
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	// Start fleet update processor
	go processFleetUpdate(ctx, c, req, stream, nodeChan, doneChan, status, jsonReq)

	// Wait for completion or timeout
	select {
	case err := <-doneChan:
		// Clean up subscription
		c.client.Unsubscribe(topicQos)
		if err != nil {
			return fmt.Errorf("fleet update failed: %w", err)
		}
		c.logger.Info().Msg("Fleet update completed successfully")
		return nil
	case <-ctx.Done():
		c.client.Unsubscribe(topicQos)
		return fmt.Errorf("fleet update timed out after %v", 5*time.Minute)
	}
}

func (c *mqtt_client) UpdateNode(req *v1.NodeUpdate, stream grpc.ServerStreamingServer[v1.UpdateResponse]) error {
	// Create channel for the stream and done signal
	streamChan := make(chan v1.UpdateState)
	doneChan := make(chan error)
	timeout := time.After(40 * time.Second) // Add a reasonable timeout

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		c.logger.Err(token.Error()).Msg("error connecting to mqtt broker")
		return token.Error()
	}

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

func (c *mqtt_client) Hello(ep *empty.Empty, stream grpc.ServerStreamingServer[v1.HelloResponse]) error {
	topic := "hello"
	messageChan := make(chan string)
	token := c.client.Subscribe(topic, 0, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		c.logger.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		messageChan <- string(msg.Payload())
	})

	token.Wait()
	go func() {
		for message := range messageChan {
			err := stream.Send(&v1.HelloResponse{
				SerialNumber: message,
			})
			if err != nil {
				return
			}
		}
	}()
	<-stream.Context().Done()
	return nil

}
func (c *mqtt_client) Log(ep *empty.Empty, stream grpc.ServerStreamingServer[v1.LogResponse]) error {

	topic := "log"
	messageChan := make(chan string)
	token := c.client.Subscribe(topic, 0, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		c.logger.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		messageChan <- string(msg.Payload())
	})
	defer c.client.Unsubscribe(topic)

	token.Wait()
	go func() {
		for message := range messageChan {
			err := stream.Send(&v1.LogResponse{
				Message: message,
			})
			if err != nil {
				return
			}
		}
	}()

	<-stream.Context().Done()
	return nil
}

/********************************** End grpc service for control_plane *******************************************/

// Define internal message structure
type nodeUpdateMessage struct {
	SerialNumber string
	State        v1.UpdateState
	TxId         string
	Timestamp    time.Time
	Error        string
}

// Prepare node tracking
type fleetStatus struct {
	mu              sync.RWMutex
	nodes           map[string]v1.UpdateState
	expectedNodes   map[string]bool
	lastGlobalState v1.UpdateState
}

// Process the fleet update and manage state transitions
func processFleetUpdate(
	ctx context.Context,
	c *mqtt_client,
	req *v1.FleetUpdate,
	stream grpc.ServerStreamingServer[v1.UpdateResponse],
	nodeChan chan nodeUpdateMessage,
	doneChan chan error,
	status *fleetStatus,
	configPayload []byte,
) {

	// Phase 1: Publish configs to all nodes
	if err := publishConfigs(ctx, c, req, status, configPayload); err != nil {
		doneChan <- err
		return
	}

	// Initial state notification
	if err := sendUpdateResponse(stream, v1.UpdateState_PUBLISHED, req.Transaction.TxId); err != nil {
		doneChan <- fmt.Errorf("failed to send initial state: %w", err)
		return
	}

	// Process node updates
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-nodeChan:
			// Process this node update
			newGlobalState, err := updateNodeStatus(c, stream, req, status, msg)
			if err != nil {
				doneChan <- err
				return
			}

			// Handle global state transitions
			if newGlobalState != v1.UpdateState_UNKNOWN {
				switch newGlobalState {
				case v1.UpdateState_UPDATE_APPLICABLE:
					// All nodes have received configs and are ready to apply
					// Phase 2: Trigger application of configs
					c.logger.Info().Msg("All nodes are ready to apply update, sending apply signal")
					if err := triggerApply(ctx, c, req, true); err != nil {
						doneChan <- err
						return
					}

				case v1.UpdateState_NODE_APPLIED:
					// All nodes have successfully applied configs
					c.logger.Info().Msg("All nodes have successfully applied the update")
					doneChan <- nil
					return

				case v1.UpdateState_ERROR:
					// At least one node failed, trigger rollback
					c.logger.Error().Str("node", msg.SerialNumber).Msg("Node reported error, triggering rollback")
					if err := triggerApply(ctx, c, req, false); err != nil {
						c.logger.Error().Err(err).Msg("Rollback failed")
					}
					doneChan <- fmt.Errorf("update failed on node %s: %s", msg.SerialNumber, msg.Error)
					return
				}
			}
		}
	}
}

// Publish configs to all nodes in the fleet
func publishConfigs(
	ctx context.Context,
	c *mqtt_client,
	req *v1.FleetUpdate,
	status *fleetStatus,
	configPayload []byte,
) error {
	c.logger.Info().Msgf("Publishing configs to %d nodes", len(req.NodeUpdateItems))

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while publishing configs")
		default:
			topicConfig := node.SerialNumber + "/sync/config"
			configToken := c.client.Publish(topicConfig, 2, false, configPayload)
			configToken.Wait()

			if err := configToken.Error(); err != nil {
				c.logger.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to publish config")
				return fmt.Errorf("failed to publish config to node %s: %w", node.SerialNumber, err)
			}

			c.logger.Debug().Str("node", node.SerialNumber).Msg("Config published")

			// Set initial state for this node
			status.mu.Lock()
			status.nodes[node.SerialNumber] = v1.UpdateState_PUBLISHED
			status.mu.Unlock()
		}
	}

	return nil
}

// Update status for a single node and check if fleet state should change
func updateNodeStatus(
	c *mqtt_client,
	stream grpc.ServerStreamingServer[v1.UpdateResponse],
	req *v1.FleetUpdate,
	status *fleetStatus,
	msg nodeUpdateMessage,
) (v1.UpdateState, error) {
	// Update node status
	status.mu.Lock()
	defer status.mu.Unlock()

	// Check if we're tracking this node
	if !status.expectedNodes[msg.SerialNumber] {
		c.logger.Warn().Str("node", msg.SerialNumber).Msg("Received update from unexpected node")
		return v1.UpdateState_UNKNOWN, nil
	}

	// Update node state
	status.nodes[msg.SerialNumber] = msg.State

	// Send individual node update
	if err := sendUpdateResponse(stream, msg.State, req.Transaction.TxId); err != nil {
		return v1.UpdateState_UNKNOWN, fmt.Errorf("failed to send update response: %w", err)
	}

	// Check if this triggers a global state change
	if msg.State == v1.UpdateState_ERROR {
		// Immediately propagate errors
		status.lastGlobalState = v1.UpdateState_ERROR
		return v1.UpdateState_ERROR, nil
	}

	// Check if all nodes have reached a specific state
	allNodesInState := func(state v1.UpdateState) bool {
		for serialNumber := range status.expectedNodes {
			nodeState, exists := status.nodes[serialNumber]
			if !exists || nodeState < state {
				return false
			}
		}
		return true
	}

	if status.lastGlobalState < v1.UpdateState_UPDATE_APPLICABLE &&
		allNodesInState(v1.UpdateState_UPDATE_APPLICABLE) {
		status.lastGlobalState = v1.UpdateState_UPDATE_APPLICABLE
		return v1.UpdateState_UPDATE_APPLICABLE, nil
	}

	if status.lastGlobalState < v1.UpdateState_NODE_APPLIED &&
		allNodesInState(v1.UpdateState_NODE_APPLIED) {
		status.lastGlobalState = v1.UpdateState_NODE_APPLIED
		return v1.UpdateState_NODE_APPLIED, nil
	}

	return v1.UpdateState_UNKNOWN, nil
}

// Trigger apply or rollback on all nodes
func triggerApply(
	ctx context.Context,
	c *mqtt_client,
	req *v1.FleetUpdate,
	apply bool,
) error {
	action := "apply"
	if !apply {
		action = "rollback"
	}

	c.logger.Info().Msgf("Triggering %s on all nodes", action)

	payload := "true"
	if !apply {
		payload = "false"
	}

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while triggering %s", action)
		default:
			topicApply := node.SerialNumber + "/sync/config"
			token := c.client.Publish(topicApply, 2, false, []byte(payload))
			token.Wait()

			if err := token.Error(); err != nil {
				c.logger.Error().Err(err).Str("node", node.SerialNumber).Msgf("Failed to trigger %s", action)
				return fmt.Errorf("failed to trigger %s on node %s: %w", action, node.SerialNumber, err)
			}

			c.logger.Debug().Str("node", node.SerialNumber).Msgf("%s triggered", action)
		}
	}

	return nil
}

func sendUpdateResponse(
	stream grpc.ServerStreamingServer[v1.UpdateResponse],
	state v1.UpdateState,
	txId string,
) error {
	return stream.Send(&v1.UpdateResponse{
		UpdateState: state,
		Timestamp:   timestamppb.New(time.Now()),
		TxId:        txId,
	})
}
