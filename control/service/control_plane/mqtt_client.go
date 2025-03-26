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
		logger:        &log,
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
		factory.client_config.AddBroker("tcp://" + cfg.Address)
	} else {
		factory.client_config.AddBroker("tls://" + cfg.Address)
		factory.client_config.CustomOpenConnectionFn = mqtt_paho.Get_custom_function(cfg.EndpointConfig)
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
	log.Debug().Str("topic", msg.Topic()).Str("payload", string(msg.Payload())).Msg("Received message")
}

var connectHandler mqtt_paho.OnConnectHandler = func(client mqtt_paho.Client) {
	log.Debug().Msg("Connected to MQTT broker")
}

var connectLostHandler mqtt_paho.ConnectionLostHandler = func(client mqtt_paho.Client, err error) {
	log.Error().Err(err).Msg("Connection lost to MQTT broker")
}

var reconHandler mqtt_paho.ReconnectHandler = func(client mqtt_paho.Client, opts *mqtt_paho.ClientOptions) {
	log.Debug().Msg("Reconnecting to MQTT broker")
}

/**********************************End  Handler *******************************************/

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

	type control_msg struct {
		Status        int32  `json:"status"`
		Module        string `json:"module"`
		Serial_number string `json:"serial_number"`
		Msg           string `json:"msg"`
	}

	serialNumber := req.NodeUpdateItem.SerialNumber

	// Use high qos
	topicQos := serialNumber + "/control/qos"
	topicConfig := serialNumber + "/config"     // Topic for initial config
	topicSync := serialNumber + "/control/sync" // Topic for control messages (fixed path)

	// Subscribe to the topic
	qosToken := c.client.Subscribe(topicQos, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		payload := msg.Payload()
		// Parse payload to v1.UpdateState
		var updateState v1.UpdateState
		var control_msg control_msg

		err := json.Unmarshal(payload, &control_msg)

		fmt.Printf("control_msg: %+v\n", control_msg)
		if err != nil {
			log.Err(err).Msg("error unmarshalling update state")
			doneChan <- err
			return
		}
		updateState = v1.UpdateState(control_msg.Status)
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
		// Unsubscribe before returning
		c.client.Unsubscribe(topicQos)
		return status.Errorf(codes.Internal, "failed to marshal request")
	}

	// Publish config to client
	configToken := c.client.Publish(topicConfig, 2, false, jsonReq)
	configToken.Wait()
	if configToken.Error() != nil {
		log.Err(configToken.Error()).Msg("error publishing update to node")
		// Unsubscribe before returning
		c.client.Unsubscribe(topicQos)
		return configToken.Error()
	}

	go func() {
		// Send initial PUBLISHED state
		err := stream.Send(&v1.UpdateResponse{
			UpdateState: v1.UpdateState_UPDATE_PUBLISHED,
			Timestamp:   timestamppb.New(time.Now()),
			TxId:        req.Transaction.TxId,
		})
		if err != nil {
			doneChan <- err
			return
		}

		// Process state updates with channel monitoring
		for {
			select {
			case <-stream.Context().Done():
				// Context cancelled (stream closed by client)
				doneChan <- stream.Context().Err()
				return
			case updateState, ok := <-streamChan:
				if !ok {
					// Channel closed
					return
				}

				// Handle state transitions
				if updateState == v1.UpdateState_UPDATE_APPLICABLE {
					// Node is ready to apply the update, send apply request
					log.Debug().Str("node", serialNumber).Msg("Node ready for update, sending apply request")
					applyToken := c.client.Publish(topicSync, 2, false, fmt.Sprintf(`{"status": %d,"tx_id":%d}`, v1.UpdateState_UPDATE_APPLY_REQ, req.Transaction.TxId))
					applyToken.Wait()
					if applyToken.Error() != nil {
						log.Err(applyToken.Error()).Msg("error sending apply request")
						doneChan <- applyToken.Error()
						return
					}
				} else if updateState == v1.UpdateState_UPDATE_APPLIED {
					// Node has applied the update, send acknowledgment
					log.Debug().Str("node", serialNumber).Msg("Node applied update, sending acknowledgment")
					ackToken := c.client.Publish(topicSync, 2, false, []byte(fmt.Sprintf(`{"status": %d,"tx_id":%d}`, v1.UpdateState_UPDATE_ACKNOWLEDGED, req.Transaction.TxId)))
					ackToken.Wait()
					if ackToken.Error() != nil {
						log.Err(ackToken.Error()).Msg("error sending acknowledgment")
						updateState = v1.UpdateState_UPDATE_ERROR
					}
					updateState = v1.UpdateState_UPDATE_APPLIED
				}

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

				fac.logger.Debug().Msgf("Update to node %s state: %s; tx_id: %d",
					serialNumber, updateState.String(), req.Transaction.TxId)

				// Terminal states
				if updateState == v1.UpdateState_UPDATE_APPLIED ||
					updateState == v1.UpdateState_UPDATE_ERROR {
					log.Info().Msgf("Update process finished for node %s state: %s; tx_id: %d",
						serialNumber, updateState.String(), req.Transaction.TxId)
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
		// Unsubscribe before returning
		c.client.Unsubscribe(topicQos)
		return err
	case <-timeout:
		// Unsubscribe before returning
		c.client.Unsubscribe(topicQos)
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

	topic := "+/control/hello"
	messageChan := make(chan string)
	token := c.client.Subscribe(topic, 0, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		// Extract serial number from topic (first element)
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) > 0 {
			serialNumber := parts[0]
			messageChan <- serialNumber
		}
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
	type logMessage struct {
		Message      string `json:"message,omitempty"`
		Level        int32  `json:"level,omitempty"`
		Module       string `json:"module,omitempty"`
		SerialNumber string `json:"serial_number,omitempty"`
	}

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
			var logMessage logMessage
			err := json.Unmarshal([]byte(message), &logMessage)
			if err != nil {
				log.Err(err).Msg("error unmarshalling log message")
				continue
			}
			err = stream.Send(&v1.LogResponse{
				Message:      logMessage.Message,
				Level:        &logMessage.Level,
				Module:       &logMessage.Module,
				SerialNumber: logMessage.SerialNumber,
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
	TxId         int32
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

/********************************** grpc service for control_plane *******************************************/
func (fac *MqttFactory) UpdateFleet(req *v1.FleetUpdate, stream grpc.ServerStreamingServer[v1.FleetResponse]) error {
	log.Info().Msgf("Starting fleet update for %d nodes, tx_id: %d", len(req.NodeUpdateItems), req.Transaction.TxId)

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
		lastGlobalState: v1.UpdateState_UPDATE_PUBLISHED,
	}

	// Register expected nodes
	for _, node := range req.NodeUpdateItems {
		status.expectedNodes[node.SerialNumber] = true
	}

	// Subscribe to all node QoS status topics - this is where we receive update state messages
	topicQos := "+/control/qos"

	// Subscribe to all node responses
	qosToken := c.client.Subscribe(topicQos, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		// Extract serial number from topic
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) < 3 {
			fac.logger.Error().Msgf("Invalid topic format: %s", msg.Topic())
			return
		}
		serialNumber := parts[0]

		// Parse update state
		var updateState v1.UpdateState
		var errorMsg string
		var txId int32

		// Try to unmarshal payload
		var payload struct {
			Status int32  `json:"status"`
			TxId   int32  `json:"tx_id"`
			Error  string `json:"error,omitempty"`
		}

		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			fac.logger.Error().Err(err).Str("payload", string(msg.Payload())).Msg("Failed to parse node response")
			return
		}

		updateState = v1.UpdateState(payload.Status)
		txId = payload.TxId
		errorMsg = payload.Error

		fac.logger.Debug().
			Str("node", serialNumber).
			Str("state", updateState.String()).
			Int32("tx_id", txId).
			Msg("Node update status received")

		// Send to processing channel
		select {
		case nodeChan <- nodeUpdateMessage{
			SerialNumber: serialNumber,
			State:        updateState,
			TxId:         txId,
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

	jsonReq, err := json.Marshal(req)
	// Prepare update payload
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

// Process the fleet update and manage state transitions
func processFleetUpdate(
	ctx context.Context,
	c *client,
	req *v1.FleetUpdate,
	stream grpc.ServerStreamingServer[v1.FleetResponse],
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

	// Send initial global state to the client
	if err := sendFleetResponse(stream, v1.UpdateState_UPDATE_PUBLISHED, req.Transaction.TxId, "Fleet update published"); err != nil {
		doneChan <- fmt.Errorf("failed to send initial state: %w", err)
		return
	}

	// Track actions taken to avoid duplicates
	applySent := false
	ackSent := false

	// Process node updates
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-nodeChan:
			// Process this node update
			newGlobalState, err := updateNodeStatus(status, msg)
			if err != nil {
				// If the error is just that we've already reported it, continue
				if err.Error() == "error already reported" {
					continue
				}
				doneChan <- err
				return
			}

			// Handle global state transitions based on the returned global state
			switch newGlobalState {
			case v1.UpdateState_UPDATE_APPLICABLE:
				// All nodes have received configs and are ready to apply
				// But only send apply request once
				if !applySent {
					log.Info().Msg("All nodes are ready to apply update, sending apply signal")

					// Send global state update to client
					if err := sendFleetResponse(stream, v1.UpdateState_UPDATE_APPLICABLE, req.Transaction.TxId, "All nodes ready for update"); err != nil {
						doneChan <- fmt.Errorf("failed to send applicable state: %w", err)
						return
					}

					if err := sendUpdateApplyRequest(ctx, c, req); err != nil {
						doneChan <- err
						return
					}

					applySent = true
				}

			case v1.UpdateState_UPDATE_APPLIED:
				// All nodes have successfully applied configs
				// But only send acknowledgement once
				if !ackSent {
					log.Info().Msg("All nodes have successfully applied the update, sending acknowledgement")

					// Send global state update to client
					if err := sendFleetResponse(stream, v1.UpdateState_UPDATE_APPLIED, req.Transaction.TxId, "All nodes have applied update"); err != nil {
						doneChan <- fmt.Errorf("failed to send applied state: %w", err)
						return
					}

					if err := sendUpdateAcknowledgement(ctx, c, req); err != nil {
						doneChan <- err
						return
					}

					ackSent = true
					doneChan <- nil
					return
				}

			case v1.UpdateState_UPDATE_ERROR:
				// Error state received
				log.Error().Str("node", msg.SerialNumber).Msg("Node reported error, ending update process")

				// Send global error state to client
				if err := sendFleetResponse(stream, v1.UpdateState_UPDATE_ERROR, req.Transaction.TxId,
					fmt.Sprintf("Update failed on node %s: %s", msg.SerialNumber, msg.Error)); err != nil {
					log.Error().Err(err).Msg("Failed to send error state")
				}

				doneChan <- fmt.Errorf("update failed on node %s: %s", msg.SerialNumber, msg.Error)
				return
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
			// Use the new topic structure
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
			status.nodes[node.SerialNumber] = v1.UpdateState_UPDATE_PUBLISHED
			status.mu.Unlock()
		}
	}

	return nil
}

// Update status for a single node and check if fleet state should change
func updateNodeStatus(
	status *fleetStatus,
	msg nodeUpdateMessage,
) (v1.UpdateState, error) {
	// Update node status
	status.mu.Lock()
	defer status.mu.Unlock()

	// Check if we're tracking this node
	if !status.expectedNodes[msg.SerialNumber] {
		log.Warn().Str("node", msg.SerialNumber).Msg("Received update from unexpected node")
		return v1.UpdateState_UPDATE_ERROR, nil
	}

	// Check if node state actually changed
	previousNodeState, exists := status.nodes[msg.SerialNumber]
	if exists && previousNodeState == msg.State {
		// Node state hasn't changed, no need to update
		log.Debug().
			Str("node", msg.SerialNumber).
			Str("state", msg.State.String()).
			Msg("Node state unchanged")
		return status.lastGlobalState, nil
	}

	// Update node state
	status.nodes[msg.SerialNumber] = msg.State

	// Log individual node updates
	log.Debug().
		Str("node", msg.SerialNumber).
		Str("state", msg.State.String()).
		Int32("tx_id", msg.TxId).
		Msg("Node state updated")

	// Check if this triggers a global state change
	if msg.State == v1.UpdateState_UPDATE_ERROR {
		// Only propagate error if we haven't already
		if status.lastGlobalState != v1.UpdateState_UPDATE_ERROR {
			log.Info().
				Str("node", msg.SerialNumber).
				Str("previous_global_state", status.lastGlobalState.String()).
				Str("new_global_state", "UPDATE_ERROR").
				Msg("Global state changing to ERROR")

			status.lastGlobalState = v1.UpdateState_UPDATE_ERROR
			return v1.UpdateState_UPDATE_ERROR, nil
		}
		// We already reported the error, no need to do it again
		return v1.UpdateState_UPDATE_ERROR, fmt.Errorf("error already reported")
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

	// Check for UPDATE_APPLICABLE transition
	if status.lastGlobalState < v1.UpdateState_UPDATE_APPLICABLE &&
		allNodesInState(v1.UpdateState_UPDATE_APPLICABLE) {
		log.Info().
			Str("previous_global_state", status.lastGlobalState.String()).
			Str("new_global_state", "UPDATE_APPLICABLE").
			Msg("Global state changing to APPLICABLE")

		status.lastGlobalState = v1.UpdateState_UPDATE_APPLICABLE
		return v1.UpdateState_UPDATE_APPLICABLE, nil
	}

	// Check for UPDATE_APPLIED transition
	if status.lastGlobalState < v1.UpdateState_UPDATE_APPLIED &&
		allNodesInState(v1.UpdateState_UPDATE_APPLIED) {
		log.Info().
			Str("previous_global_state", status.lastGlobalState.String()).
			Str("new_global_state", "UPDATE_APPLIED").
			Msg("Global state changing to APPLIED")

		status.lastGlobalState = v1.UpdateState_UPDATE_APPLIED
		return v1.UpdateState_UPDATE_APPLIED, nil
	}

	// No change to global state
	return status.lastGlobalState, nil
}

// Send UPDATE_APPLY_REQ to all nodes
func sendUpdateApplyRequest(
	ctx context.Context,
	c *client,
	req *v1.FleetUpdate,
) error {
	log.Info().Msg("Sending apply request to all nodes")

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while sending apply request")
		default:
			// Use the new topic structure for sync control messages
			topicSync := node.SerialNumber + "/control/sync"

			// Create payload with status and transaction ID
			payload := fmt.Sprintf(`{"status": %d, "tx_id": %d}`,
				v1.UpdateState_UPDATE_APPLY_REQ,
				req.Transaction.TxId)

			token := c.client.Publish(topicSync, 2, false, []byte(payload))
			token.Wait()

			if err := token.Error(); err != nil {
				log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to send apply request")
				return fmt.Errorf("failed to send apply request to node %s: %w", node.SerialNumber, err)
			}

			log.Debug().Str("node", node.SerialNumber).Msg("Apply request sent")
		}
	}

	return nil
}

// Send UPDATE_ACKNOWLEDGED to all nodes
func sendUpdateAcknowledgement(
	ctx context.Context,
	c *client,
	req *v1.FleetUpdate,
) error {
	log.Info().Msg("Sending acknowledgement to all nodes")

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while sending acknowledgement")
		default:
			// Use the new topic structure for sync control messages
			topicSync := node.SerialNumber + "/control/sync"

			// Create payload with status and transaction ID
			payload := fmt.Sprintf(`{"status": %d, "tx_id": %d}`,
				v1.UpdateState_UPDATE_ACKNOWLEDGED,
				req.Transaction.TxId)

			token := c.client.Publish(topicSync, 2, false, []byte(payload))
			token.Wait()

			if err := token.Error(); err != nil {
				log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to send acknowledgement")
				return fmt.Errorf("failed to send acknowledgement to node %s: %w", node.SerialNumber, err)
			}

			log.Debug().Str("node", node.SerialNumber).Msg("Acknowledgement sent")
		}
	}

	return nil
}

// Send FleetResponse to client - only for global state changes
func sendFleetResponse(
	stream grpc.ServerStreamingServer[v1.FleetResponse],
	state v1.UpdateState,
	txId int32,
	meta string,
) error {
	return stream.Send(&v1.FleetResponse{
		UpdateState: state,
		Timestamp:   timestamppb.New(time.Now()),
		TxId:        txId,
		Meta:        meta,
	})
}

// marshalFleetUpdateWithEnums handles custom marshalling for FleetUpdate
// to properly set numeric values for enum fields like ProxyType
func marshalFleetUpdateWithEnums(req *v1.FleetUpdate) ([]byte, error) {
	// Create a copy that will be properly marshalled
	type proxyWithNumericType struct {
		Name               string `json:"name,omitempty"`
		ServerEndpointAddr string `json:"server_endpoint_addr,omitempty"`
		ClientEndpointAddr string `json:"client_endpoint_addr,omitempty"`
		ProxyType          int32  `json:"proxy_type"`
	}

	type groupProxyUpdateWithEnums struct {
		GroupName      string                 `json:"group_name,omitempty"`
		EndpointConfig *v1.EndpointConfig     `json:"endpoint_config,omitempty"`
		LegacyConfig   *v1.EndpointConfig     `json:"legacy_config,omitempty"`
		GroupLogLevel  int32                  `json:"group_log_level,omitempty"`
		Proxies        []proxyWithNumericType `json:"proxies,omitempty"`
	}

	type nodeUpdateItemWithEnums struct {
		SerialNumber     string                      `json:"serial_number,omitempty"`
		NetworkIndex     int32                       `json:"network_index,omitempty"`
		Locality         string                      `json:"locality,omitempty"`
		VersionSetId     string                      `json:"version_set_id,omitempty"`
		GroupProxyUpdate []groupProxyUpdateWithEnums `json:"group_proxy_update,omitempty"`
	}

	type fleetUpdateWithEnums struct {
		Transaction     *v1.Transaction           `json:"transaction,omitempty"`
		NodeUpdateItems []nodeUpdateItemWithEnums `json:"node_update_items,omitempty"`
	}

	// Convert the request to our custom structure
	customReq := fleetUpdateWithEnums{
		Transaction: req.Transaction,
	}

	if len(req.NodeUpdateItems) > 0 {
		customReq.NodeUpdateItems = make([]nodeUpdateItemWithEnums, len(req.NodeUpdateItems))

		for i, nodeItem := range req.NodeUpdateItems {
			customNodeItem := nodeUpdateItemWithEnums{
				SerialNumber: nodeItem.SerialNumber,
				NetworkIndex: nodeItem.NetworkIndex,
				Locality:     nodeItem.Locality,
				VersionSetId: nodeItem.VersionSetId,
			}

			// Handle GroupProxyUpdate
			if len(nodeItem.GroupProxyUpdate) > 0 {
				customNodeItem.GroupProxyUpdate = make([]groupProxyUpdateWithEnums, len(nodeItem.GroupProxyUpdate))

				for j, gpu := range nodeItem.GroupProxyUpdate {
					customGpu := groupProxyUpdateWithEnums{
						GroupName:      gpu.GroupName,
						EndpointConfig: gpu.EndpointConfig,
						LegacyConfig:   gpu.LegacyConfig,
						GroupLogLevel:  gpu.GroupLogLevel,
					}

					// Handle proxies with explicit enum type
					if len(gpu.Proxies) > 0 {
						customGpu.Proxies = make([]proxyWithNumericType, len(gpu.Proxies))

						for k, proxy := range gpu.Proxies {
							customGpu.Proxies[k] = proxyWithNumericType{
								Name:               proxy.Name,
								ServerEndpointAddr: proxy.ServerEndpointAddr,
								ClientEndpointAddr: proxy.ClientEndpointAddr,
								ProxyType:          int32(proxy.ProxyType), // Convert enum to int32
							}
						}
					}

					customNodeItem.GroupProxyUpdate[j] = customGpu
				}
			}

			customReq.NodeUpdateItems[i] = customNodeItem
		}
	}

	// Marshal with the custom structure that properly handles enum values
	return json.Marshal(customReq)
}
