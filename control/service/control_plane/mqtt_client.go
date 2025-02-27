package control_plane

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	// "github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	mqtt_paho "github.com/Laboratory-for-Safe-and-Secure-Systems/paho.mqtt.golang"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type client struct {
	id_name string
	client  mqtt_paho.Client
	subs    []string
}

type MqttFactory struct {
	mu            sync.Mutex
	client_config *mqtt_paho.ClientOptions
	cfg           types.ControlPlaneConfig
	logger        *zerolog.Logger
	clients       []*client
	v1.UnimplementedControlPlaneServer
}

var log = zerolog.New(os.Stdout).Level(zerolog.DebugLevel)

func ControlPlaneInit(cfg types.ControlPlaneConfig) *MqttFactory {

	var factory *MqttFactory

	client_opts := mqtt_paho.NewClientOptions()
	factory = &MqttFactory{
		client_config: client_opts,
		cfg:           cfg,
		mu:            sync.Mutex{},
		clients:       make([]*client, 4),
	}
	factory.clients[0] = &client{
		id_name: "log",
	}
	factory.clients[1] = &client{
		id_name: "hello",
	}
	factory.clients[2] = &client{
		id_name: "update_fleet",
	}
	factory.clients[3] = &client{
		id_name: "update_node",
	}

	factory.client_config = factory.client_config.SetCleanSession(true)
	factory.client_config.SetDefaultPublishHandler(messagePubHandler)
	factory.client_config.OnConnect = connectHandler
	factory.client_config.OnConnectionLost = connectLostHandler
	factory.client_config.OnReconnecting = reconHandler

	if cfg.TcpOnly {
		factory.client_config.AddBroker("tls://" + cfg.Address)
		factory.client_config.CustomOpenConnectionFn = mqtt_paho.Get_custom_function(cfg.EndpointConfig)
	} else {
		factory.client_config.AddBroker("tcp://" + cfg.Address)
	}

	factory.client_config.SetProtocolVersion(3)
	return factory

}

func (c *client) cleanup() {
	for _, sub := range c.subs {
		c.client.Unsubscribe(sub)
	}
	c.client.Disconnect(250)

	//clean subs
	c.subs = c.subs[:0]

}

func (f *MqttFactory) GetClient(id string) (*client, error) {
	for _, c := range f.clients {
		if c.id_name == id {
			f.client_config.SetClientID(id)
			c.client = mqtt_paho.NewClient(f.client_config)
			// Connect with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			connectChan := make(chan error, 1)
			go func() {
				token := c.client.Connect()
				token.Wait()
				connectChan <- token.Error()
			}()

			select {
			case err := <-connectChan:
				if err != nil {
					return nil, fmt.Errorf("failed to connect MQTT client: %w", err)
				}
				return c, nil
			case <-ctx.Done():
				return nil, fmt.Errorf("connection timeout")
			}

		}
	}
	return nil, fmt.Errorf("client not found")
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

/********************************** grpc service for control_plane *******************************************/
func (fac *MqttFactory) UpdateFleet(req *v1.FleetUpdate, stream grpc.ServerStreamingServer[v1.UpdateResponse]) error {
	fac.logger.Info().Msgf("Starting fleet update for %d nodes, tx_id: %s", len(req.NodeUpdateItems), req.Transaction.TxId)

	c, err := fac.GetClient("update_fleet")
	if err != nil {
		log.Err(err).Msg("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

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
			fac.logger.Error().Msgf("Invalid topic format: %s", msg.Topic())
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
				fac.logger.Error().Err(err).Str("payload", string(msg.Payload())).Msg("Failed to parse node response")
				return
			}
		}

		fac.logger.Debug().
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
	c.subs = append(c.subs, topicQos)

	if err := qosToken.Error(); err != nil {
		return fmt.Errorf("failed to subscribe to node responses: %w", err)
	}
	qosToken.Wait()

	// Prepare update payload
	jsonReq, err := json.Marshal(req)
	if err != nil {
		fac.logger.Error().Err(err).Msg("Failed to marshal update request")
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
		log.Info().Msg("Fleet update completed successfully")
		return nil
	case <-ctx.Done():
		c.client.Unsubscribe(topicQos)
		return fmt.Errorf("fleet update timed out after %v", 5*time.Minute)
	}
}

func (fac *MqttFactory) UpdateNode(req *v1.NodeUpdate, stream grpc.ServerStreamingServer[v1.UpdateResponse]) error {
	// Create channel for the stream and done signal
	streamChan := make(chan v1.UpdateState)
	doneChan := make(chan error)
	timeout := time.After(40 * time.Second) // Add a reasonable timeout
	c, err := fac.GetClient("update_node")
	if err != nil {
		log.Err(err).Msg("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	serialNumber := req.NodeUpdateItem.SerialNumber
	// Use high qos
	topicQos := serialNumber + "/control/qos"
	topicConfig := serialNumber + "/config" // Fixed the topic path

	// Subscribe to the topic
	qosToken := c.client.Subscribe(topicQos, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		payload := msg.Payload()
		// Parse payload to v1.UpdateState
		var updateState v1.UpdateState
		err := json.Unmarshal(payload, &updateState)
		if err != nil {
			log.Err(err).Msg("error unmarshalling update state")
			doneChan <- err
			return
		}
		streamChan <- updateState
	})
	c.subs = append(c.subs, topicQos)

	if err := qosToken.Error(); err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to qos topic")
	}
	qosToken.Wait()

	jsonReq, err := json.Marshal(req)
	if err != nil {
		log.Err(err).Msg("error marshalling request")
		return status.Errorf(codes.Internal, "failed to marshal request")
	}

	// Publish config to client
	configToken := c.client.Publish(topicConfig, 2, false, jsonReq)
	configToken.Wait()
	if configToken.Error() != nil {
		log.Err(configToken.Error()).Msg("error publishing update to node")
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

				log.Debug().Msgf("Update to node %s state: %s; tx_id: %s",
					serialNumber, updateState.String(), req.Transaction.TxId)

				// Terminal states
				if updateState == v1.UpdateState_NODE_APPLIED ||
					updateState == v1.UpdateState_ERROR {
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
		return fmt.Errorf("update operation timed out for node %s", serialNumber)
	}
}

func (fac *MqttFactory) Hello(ep *empty.Empty, stream grpc.ServerStreamingServer[v1.HelloResponse]) error {

	c, err := fac.GetClient("hello")
	if err != nil {
		log.Err(err).Msg("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	topic := "hello"
	messageChan := make(chan string)
	token := c.client.Subscribe(topic, 0, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		messageChan <- string(msg.Payload())
	})
	c.subs = append(c.subs, topic)
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
func (fac *MqttFactory) Log(ep *empty.Empty, stream grpc.ServerStreamingServer[v1.LogResponse]) error {
	c, err := fac.GetClient("log")
	if err != nil {
		log.Err(err).Msgf("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	topic := "log"
	messageChan := make(chan string, 3)
	token := c.client.Subscribe(topic, 0, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		messageChan <- string(msg.Payload())
	})
	c.subs = append(c.subs, topic)

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
	c *client,
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
					log.Info().Msg("All nodes are ready to apply update, sending apply signal")
					if err := triggerApply(ctx, c, req, true); err != nil {
						doneChan <- err
						return
					}

				case v1.UpdateState_NODE_APPLIED:
					// All nodes have successfully applied configs
					log.Info().Msg("All nodes have successfully applied the update")
					doneChan <- nil
					return

				case v1.UpdateState_ERROR:
					// At least one node failed, trigger rollback
					log.Error().Str("node", msg.SerialNumber).Msg("Node reported error, triggering rollback")
					if err := triggerApply(ctx, c, req, false); err != nil {
						log.Error().Err(err).Msg("Rollback failed")
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
	c *client,
	req *v1.FleetUpdate,
	status *fleetStatus,
	configPayload []byte,
) error {
	log.Info().Msgf("Publishing configs to %d nodes", len(req.NodeUpdateItems))

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while publishing configs")
		default:
			topicConfig := node.SerialNumber + "/sync/config"
			configToken := c.client.Publish(topicConfig, 2, false, configPayload)
			configToken.Wait()

			if err := configToken.Error(); err != nil {
				log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to publish config")
				return fmt.Errorf("failed to publish config to node %s: %w", node.SerialNumber, err)
			}

			log.Debug().Str("node", node.SerialNumber).Msg("Config published")

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
	c *client,
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
		log.Warn().Str("node", msg.SerialNumber).Msg("Received update from unexpected node")
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
	c *client,
	req *v1.FleetUpdate,
	apply bool,
) error {
	action := "apply"
	if !apply {
		action = "rollback"
	}

	log.Info().Msgf("Triggering %s on all nodes", action)

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
				log.Error().Err(err).Str("node", node.SerialNumber).Msgf("Failed to trigger %s", action)
				return fmt.Errorf("failed to trigger %s on node %s: %w", action, node.SerialNumber, err)
			}

			log.Debug().Str("node", node.SerialNumber).Msgf("%s triggered", action)
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
