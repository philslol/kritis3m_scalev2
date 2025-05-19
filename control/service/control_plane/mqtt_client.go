package control_plane

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	grpc_controlplane "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/control_plane"
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	mqtt_paho "github.com/Laboratory-for-Safe-and-Secure-Systems/paho.mqtt.golang"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	clients       []*client
	grpc_controlplane.UnimplementedControlPlaneServer
}

var mqtt_log zerolog.Logger

func ControlPlaneInit(cfg types.ControlPlaneConfig) *MqttFactory {

	var factory *MqttFactory
	mqtt_log = types.CreateLogger("mqtt_client", cfg.Log.Level, cfg.Log.File)

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
	mqtt_log.Debug().Str("topic", msg.Topic()).Str("payload", string(msg.Payload())).Msg("Received message")
}

var connectHandler mqtt_paho.OnConnectHandler = func(client mqtt_paho.Client) {
	mqtt_log.Debug().Msg("Connected to MQTT broker")
}

var connectLostHandler mqtt_paho.ConnectionLostHandler = func(client mqtt_paho.Client, err error) {
	mqtt_log.Error().Err(err).Msg("Connection lost to MQTT broker")
}

var reconHandler mqtt_paho.ReconnectHandler = func(client mqtt_paho.Client, opts *mqtt_paho.ClientOptions) {
	mqtt_log.Debug().Msg("Reconnecting to MQTT broker")
}

/**********************************End  Handler *******************************************/

func (fac *MqttFactory) SendCertificateRequest(ctx context.Context, req *grpc_controlplane.CertificateRequest) (*grpc_controlplane.CertificateResponse, error) {
	// check req
	if req.CertType != grpc_southbound.CertType_CONTROLPLANE && req.CertType != grpc_southbound.CertType_DATAPLANE {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cert type")
	}
	//check serial number
	if req.SerialNumber == "" {
		return nil, status.Errorf(codes.InvalidArgument, "serial number is empty")
	}

	if req.HostName == "" && req.IpAddr == "" { //|| req.Port == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "host name or ip addr and port are empty")
	}

	// create topic
	topic := req.SerialNumber + "/control/cert_req"

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal request")
	}

	// get client
	c, err := fac.GetClient("update_node")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	// publish
	token := c.client.Publish(topic, 2, false, payload)
	token.Wait()
	if token.Error() != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish request")
	}

	return &grpc_controlplane.CertificateResponse{
		Retcode: 0,
	}, nil
}

func (fac *MqttFactory) UpdateNode(req *grpc_controlplane.NodeUpdate, stream grpc.ServerStreamingServer[grpc_controlplane.UpdateResponse]) error {
	// Create channel for the stream and done signal
	streamChan := make(chan grpc_controlplane.UpdateState)
	doneChan := make(chan error)
	timeout := time.After(40 * time.Second) // Add a reasonable timeout
	c, err := fac.GetClient("update_node")
	if err != nil {
		mqtt_log.Err(err).Msg("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	serialNumber := req.NodeUpdateItem.SerialNumber

	// Use high qos
	topicState := serialNumber + "/control/state"
	topicConfig := serialNumber + "/config"     // Topic for initial config
	topicSync := serialNumber + "/control/sync" // Topic for control messages (fixed path)

	// Subscribe to the topic
	qosToken := c.client.Subscribe(topicState, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		mqtt_log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		payload := msg.Payload()
		// Parse payload to v1.UpdateState
		var updateState grpc_controlplane.UpdateState
		var control_msg control_msg

		err := json.Unmarshal(payload, &control_msg)

		fmt.Printf("control_msg: %+v\n", control_msg)
		if err != nil {
			mqtt_log.Err(err).Msg("error unmarshalling update state")
			doneChan <- err
			return
		}
		updateState = grpc_controlplane.UpdateState(control_msg.Status)
		streamChan <- updateState
	})
	c.subs = append(c.subs, topicState)

	if err := qosToken.Error(); err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to qos topic")
	}
	qosToken.Wait()

	jsonReq, err := json.Marshal(req.NodeUpdateItem)
	if err != nil {
		mqtt_log.Err(err).Msg("error marshalling request")
		// Unsubscribe before returning
		c.client.Unsubscribe(topicState)
		return status.Errorf(codes.Internal, "failed to marshal request")
	}

	// Publish config to client
	configToken := c.client.Publish(topicConfig, 2, false, jsonReq)
	configToken.Wait()
	if configToken.Error() != nil {
		mqtt_log.Err(configToken.Error()).Msg("error publishing update to node")
		// Unsubscribe before returning
		c.client.Unsubscribe(topicSync)
		return configToken.Error()
	}

	go func() {
		// Send initial PUBLISHED state
		err := stream.Send(&grpc_controlplane.UpdateResponse{
			UpdateState: grpc_controlplane.UpdateState_UPDATE_PUBLISHED,
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
				if updateState == grpc_controlplane.UpdateState_UPDATE_APPLICABLE {
					// Node is ready to apply the update, send apply request
					mqtt_log.Debug().Str("node", serialNumber).Msg("Node ready for update, sending apply request")
					applyToken := c.client.Publish(topicSync, 2, false, fmt.Sprintf(`{"status": %d,"tx_id":%d}`, grpc_controlplane.UpdateState_UPDATE_APPLY_REQ, req.Transaction.TxId))
					applyToken.Wait()
					if applyToken.Error() != nil {
						mqtt_log.Err(applyToken.Error()).Msg("error sending apply request")
						doneChan <- applyToken.Error()
						return
					}
				} else if updateState == grpc_controlplane.UpdateState_UPDATE_APPLIED {
					// Node has applied the update, send acknowledgment
					mqtt_log.Debug().Str("node", serialNumber).Msg("Node applied update, sending acknowledgment")
					ackToken := c.client.Publish(topicSync, 2, false, []byte(fmt.Sprintf(`{"status": %d,"tx_id":%d}`, grpc_controlplane.UpdateState_UPDATE_ACKNOWLEDGED, req.Transaction.TxId)))
					ackToken.Wait()
					if ackToken.Error() != nil {
						mqtt_log.Err(ackToken.Error()).Msg("error sending acknowledgment")
						updateState = grpc_controlplane.UpdateState_UPDATE_ERROR
					}
					updateState = grpc_controlplane.UpdateState_UPDATE_APPLIED
				}

				// Send update to gRPC stream
				err := stream.Send(&grpc_controlplane.UpdateResponse{
					UpdateState: updateState,
					Timestamp:   timestamppb.New(time.Now()),
					TxId:        req.Transaction.TxId,
				})
				if err != nil {
					doneChan <- err
					return
				}

				mqtt_log.Debug().Msgf("Update to node %s state: %s; tx_id: %d",
					serialNumber, updateState.String(), req.Transaction.TxId)

				// Terminal states
				if updateState == grpc_controlplane.UpdateState_UPDATE_APPLIED ||
					updateState == grpc_controlplane.UpdateState_UPDATE_ERROR {
					mqtt_log.Info().Msgf("Update process finished for node %s state: %s; tx_id: %d",
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
		c.client.Unsubscribe(topicState)
		return err
	case <-timeout:
		// Unsubscribe before returning
		c.client.Unsubscribe(topicState)
		return fmt.Errorf("update operation timed out for node %s", serialNumber)
	}
}

func (fac *MqttFactory) Hello(ep *empty.Empty, stream grpc.ServerStreamingServer[grpc_controlplane.HelloResponse]) error {

	c, err := fac.GetClient("hello")
	if err != nil {
		mqtt_log.Err(err).Msg("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	topic := "+/control/hello"
	messageChan := make(chan string)
	token := c.client.Subscribe(topic, 0, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		mqtt_log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
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
			err := stream.Send(&grpc_controlplane.HelloResponse{
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
func (fac *MqttFactory) Log(ep *empty.Empty, stream grpc.ServerStreamingServer[grpc_controlplane.LogResponse]) error {
	c, err := fac.GetClient("log")
	if err != nil {
		mqtt_log.Err(err).Msgf("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	type logMessage struct {
		Timestamp int64  `json:"timestamp"`
		Module    string `json:"module"`
		Level     int32  `json:"level"`
		Message   string `json:"message"`
	}

	topic := "+/log"
	messageChan := make(chan struct {
		serialNumber string
		payload      string
	}, 3)
	token := c.client.Subscribe(topic, 0, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		mqtt_log.Debug().Msgf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		// Extract serial number from topic (first element)
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) > 0 {
			serialNumber := parts[0]
			messageChan <- struct {
				serialNumber string
				payload      string
			}{
				serialNumber: serialNumber,
				payload:      string(msg.Payload()),
			}
		}
	})
	c.subs = append(c.subs, topic)

	token.Wait()
	go func() {
		for msg := range messageChan {
			// Parse the array of log messages
			var logMessages []logMessage
			err := json.Unmarshal([]byte(msg.payload), &logMessages)
			if err != nil {
				mqtt_log.Err(err).Msg("error unmarshalling log message")
				continue
			}

			// Send each log message in the array
			for _, logMsg := range logMessages {
				err = stream.Send(&grpc_controlplane.LogResponse{
					Message:      logMsg.Message,
					Level:        &logMsg.Level,
					Module:       &logMsg.Module,
					SerialNumber: msg.serialNumber,
				})
				if err != nil {
					return
				}
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
	State        grpc_controlplane.UpdateState
	TxId         int32
	Timestamp    time.Time
	Error        string
}

// Prepare node tracking
type fleetStatus struct {
	mu              sync.RWMutex
	nodes           map[string]grpc_controlplane.UpdateState
	expectedNodes   map[string]bool
	lastGlobalState grpc_controlplane.UpdateState
}

type control_msg struct {
	Status        int32  `json:"status"`
	Module        string `json:"module"`
	Serial_number string `json:"serial_number"`
	Msg           string `json:"msg"`
	TxId          int32  `json:"tx_id,omitempty"`
}

/********************************** grpc service for control_plane *******************************************/
func (fac *MqttFactory) UpdateFleet(req *grpc_controlplane.FleetUpdate, stream grpc.ServerStreamingServer[grpc_controlplane.FleetResponse]) error {
	mqtt_log.Info().Msgf("Starting fleet update for %d nodes, tx_id: %d", len(req.NodeUpdateItems), req.Transaction.TxId)

	c, err := fac.GetClient("update_fleet")

	if err != nil {
		mqtt_log.Err(err).Msg("failed to get client")
		return status.Errorf(codes.Internal, "failed to get client")
	}
	defer c.cleanup()

	// Create communication channels
	nodeChan := make(chan nodeUpdateMessage, len(req.NodeUpdateItems)*4) // Buffer for multiple messages per node
	doneChan := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // More reasonable timeout for fleet updates
	defer cancel()

	// Initialize fleet status tracker with correct initial state
	status := &fleetStatus{
		nodes:           make(map[string]grpc_controlplane.UpdateState),
		expectedNodes:   make(map[string]bool),
		lastGlobalState: grpc_controlplane.UpdateState_UPDATE_PUBLISHED,
	}

	// Register expected nodes
	for _, node := range req.NodeUpdateItems {
		status.expectedNodes[node.SerialNumber] = true
	}

	// Subscribe to all node QoS status topics - this is where we receive update state messages
	topicState := "+/control/state"

	// Subscribe to all node responses
	qosToken := c.client.Subscribe(topicState, 2, func(client mqtt_paho.Client, msg mqtt_paho.Message) {
		// Extract serial number from topic
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) < 3 {
			mqtt_log.Error().Msgf("Invalid topic format: %s", msg.Topic())
			return
		}
		serialNumber := parts[0]

		// Parse payload to get update state
		var control_msg control_msg
		payload := msg.Payload()
		err := json.Unmarshal(payload, &control_msg)

		if err != nil {
			mqtt_log.Err(err).Msg("error unmarshalling update state")
			return
		}

		updateState := grpc_controlplane.UpdateState(control_msg.Status)
		txId := control_msg.TxId
		errorMsg := control_msg.Msg

		// Send to processing channel
		nodeChan <- nodeUpdateMessage{
			SerialNumber: serialNumber,
			State:        updateState,
			TxId:         txId,
			Timestamp:    time.Now(),
			Error:        errorMsg,
		}

		mqtt_log.Debug().
			Str("node", serialNumber).
			Str("state", updateState.String()).
			Int32("tx_id", txId).
			Msg("Received node state update")
	})
	c.subs = append(c.subs, topicState)

	if err := qosToken.Error(); err != nil {
		return fmt.Errorf("failed to subscribe to node responses: %w", err)
	}
	qosToken.Wait()

	// Start fleet update processor
	go processFleetUpdate(ctx, c, req, stream, nodeChan, doneChan, status)

	// Wait for completion or timeout
	select {
	case err := <-doneChan:
		// Clean up subscription
		c.client.Unsubscribe(topicState)
		close(nodeChan) // Close channel to prevent goroutine leaks
		//close stream

		if err != nil {
			return fmt.Errorf("fleet update failed: %w", err)
		}
		log.Info().Msg("Fleet update completed successfully")
		return nil
	case <-ctx.Done():
		c.client.Unsubscribe(topicState)
		close(nodeChan) // Close channel to prevent goroutine leaks
		return fmt.Errorf("fleet update timed out after %v", 5*time.Minute)
	}
}

// Process the fleet update and manage state transitions
func processFleetUpdate(
	ctx context.Context,
	c *client,
	req *grpc_controlplane.FleetUpdate,
	stream grpc.ServerStreamingServer[grpc_controlplane.FleetResponse],
	nodeChan chan nodeUpdateMessage,
	doneChan chan error,
	status *fleetStatus,
) {

	// Phase 1: Publish configs to all nodes
	if err := publishConfigs(ctx, c, req, status); err != nil {
		doneChan <- err
		return
	}

	// Send initial global state to the client
	if err := sendFleetResponse(stream, grpc_controlplane.UpdateState_UPDATE_PUBLISHED, req.Transaction.TxId, "Fleet update published"); err != nil {
		doneChan <- fmt.Errorf("failed to send initial state: %w", err)
		return
	}

	// Initialize the global state
	status.mu.Lock()
	status.lastGlobalState = grpc_controlplane.UpdateState_UPDATE_PUBLISHED
	status.mu.Unlock()

	// Track actions taken to avoid duplicates
	applySent := false
	ackSent := false
	rollbackSent := false

	// Process node updates
	for {
		select {
		case <-ctx.Done():
			doneChan <- fmt.Errorf("context deadline exceeded or canceled")
			return
		case msg := <-nodeChan:
			// Send individual node status update via gRPC
			if err := stream.Send(&grpc_controlplane.FleetResponse{
				UpdateState:  msg.State,
				Timestamp:    timestamppb.New(msg.Timestamp),
				TxId:         msg.TxId,
				SerialNumber: msg.SerialNumber,
				Meta:         &msg.Error,
			}); err != nil {
				doneChan <- fmt.Errorf("failed to send node status update: %w", err)
				return
			}

			// Process this node update for global state
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
			case grpc_controlplane.UpdateState_UPDATE_APPLICABLE:
				// All nodes have received configs and are ready to apply
				// But only send apply request once
				if !applySent {
					mqtt_log.Info().Msg("All nodes are ready to apply update, sending apply signal")

					// Send global state update to client
					if err := sendFleetResponse(stream, grpc_controlplane.UpdateState_UPDATE_APPLICABLE, req.Transaction.TxId, "All nodes ready for update"); err != nil {
						doneChan <- fmt.Errorf("failed to send applicable state: %w", err)
						return
					}

					// Update global state to APPLY_REQ before sending the request
					status.mu.Lock()
					status.lastGlobalState = grpc_controlplane.UpdateState_UPDATE_APPLY_REQ
					status.mu.Unlock()

					if err := sendUpdateApplyRequest(ctx, c, req); err != nil {
						doneChan <- err
						return
					}

					applySent = true
				}

			case grpc_controlplane.UpdateState_UPDATE_APPLIED:
				// All nodes have successfully applied configs
				// But only send acknowledgement once
				if !ackSent {
					mqtt_log.Info().Msg("All nodes have successfully applied the update, sending acknowledgement")

					// Send global state update to client
					if err := sendFleetResponse(stream, grpc_controlplane.UpdateState_UPDATE_APPLIED, req.Transaction.TxId, "All nodes have applied update"); err != nil {
						doneChan <- fmt.Errorf("failed to send applied state: %w", err)
						return
					}

					// Update global state to ACKNOWLEDGED before sending the acknowledgement
					status.mu.Lock()
					status.lastGlobalState = grpc_controlplane.UpdateState_UPDATE_ACKNOWLEDGED
					status.mu.Unlock()

					if err := sendUpdateAcknowledgement(ctx, c, req); err != nil {
						doneChan <- err
						return
					}

					ackSent = true
					mqtt_log.Info().Msg("Fleet update completed successfully")
					doneChan <- nil
					return // Return here to properly exit the function
				}

			case grpc_controlplane.UpdateState_UPDATE_ERROR:
				// Error state received - implement rollback
				if !rollbackSent {
					mqtt_log.Error().Str("node", msg.SerialNumber).Str("error", msg.Error).Msg("Node reported error, initiating rollback")

					// Send global error state to client
					if err := sendFleetResponse(stream, grpc_controlplane.UpdateState_UPDATE_ERROR, req.Transaction.TxId,
						fmt.Sprintf("Update failed on node %s: %s - initiating rollback", msg.SerialNumber, msg.Error)); err != nil {
						mqtt_log.Error().Err(err).Msg("Failed to send error state")
					}

					// Update global state to ROLLBACK before sending the rollback request
					status.mu.Lock()
					status.lastGlobalState = grpc_controlplane.UpdateState_UPDATE_ROLLBACK
					status.mu.Unlock()

					// Send rollback request to all nodes
					if err := sendRollbackRequest(ctx, c, req); err != nil {
						mqtt_log.Error().Err(err).Msg("Failed to send rollback request")
					}

					rollbackSent = true
					doneChan <- fmt.Errorf("update failed on node %s: %s - rollback initiated", msg.SerialNumber, msg.Error)
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
	req *grpc_controlplane.FleetUpdate,
	status *fleetStatus) error {
	mqtt_log.Info().Msgf("Publishing configs to %d nodes", len(req.NodeUpdateItems))

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while publishing configs")
		default:
			payload, err := json.Marshal(node)
			if err != nil {
				mqtt_log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to marshal node")
				return fmt.Errorf("failed to marshal node %s: %w", node.SerialNumber, err)
			}
			// Use the new topic structure
			topicConfig := node.SerialNumber + "/config"
			configToken := c.client.Publish(topicConfig, 2, false, payload)
			configToken.Wait()

			if err := configToken.Error(); err != nil {
				mqtt_log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to publish config")
				return fmt.Errorf("failed to publish config to node %s: %w", node.SerialNumber, err)
			}

			mqtt_log.Debug().Str("node", node.SerialNumber).Msg("Config published")

			// Set initial state for this node
			status.mu.Lock()
			status.nodes[node.SerialNumber] = grpc_controlplane.UpdateState_UPDATE_PUBLISHED
			status.mu.Unlock()
		}
	}

	return nil
}

// Update status for a single node and check if fleet state should change
func updateNodeStatus(
	status *fleetStatus,
	msg nodeUpdateMessage,
) (grpc_controlplane.UpdateState, error) {
	// Update node status
	status.mu.Lock()
	defer status.mu.Unlock()

	// Check if we're tracking this node
	if !status.expectedNodes[msg.SerialNumber] {
		mqtt_log.Warn().Str("node", msg.SerialNumber).Msg("Received update from unexpected node")
		return grpc_controlplane.UpdateState_UPDATE_ERROR, nil
	}

	// Check if node state actually changed
	previousNodeState, exists := status.nodes[msg.SerialNumber]
	if exists && previousNodeState == msg.State {
		// Node state hasn't changed, no need to update
		mqtt_log.Debug().
			Str("node", msg.SerialNumber).
			Str("state", msg.State.String()).
			Msg("Node state unchanged")
		return status.lastGlobalState, nil
	}

	// Update node state
	status.nodes[msg.SerialNumber] = msg.State

	// update node status

	// Log individual node updates
	mqtt_log.Debug().
		Str("node", msg.SerialNumber).
		Str("state", msg.State.String()).
		Int32("tx_id", msg.TxId).
		Msg("Node state updated")

	// Check if this triggers a global state change
	if msg.State == grpc_controlplane.UpdateState_UPDATE_ERROR {
		// Only propagate error if we haven't already
		if status.lastGlobalState != grpc_controlplane.UpdateState_UPDATE_ERROR {
			mqtt_log.Info().
				Str("node", msg.SerialNumber).
				Str("previous_global_state", status.lastGlobalState.String()).
				Str("new_global_state", "UPDATE_ERROR").
				Str("error", msg.Error).
				Msg("Global state changing to ERROR due to node error")

			status.lastGlobalState = grpc_controlplane.UpdateState_UPDATE_ERROR
			return grpc_controlplane.UpdateState_UPDATE_ERROR, nil
		}
		// We already reported the error, no need to do it again
		return grpc_controlplane.UpdateState_UPDATE_ERROR, fmt.Errorf("error already reported")
	}

	// Check if all nodes have reached a specific state
	allNodesInState := func(state grpc_controlplane.UpdateState) bool {
		for serialNumber := range status.expectedNodes {
			nodeState, exists := status.nodes[serialNumber]
			if !exists {
				return false
			}

			// For state progression check, we need to compare based on the expected sequence
			// not the raw enum values
			switch state {
			case grpc_controlplane.UpdateState_UPDATE_APPLICABLE:
				// For APPLICABLE state, nodes must be in APPLICABLE state
				if nodeState != grpc_controlplane.UpdateState_UPDATE_APPLICABLE {
					return false
				}
			case grpc_controlplane.UpdateState_UPDATE_APPLIED:
				// For APPLIED state, nodes must be in APPLIED state
				if nodeState != grpc_controlplane.UpdateState_UPDATE_APPLIED {
					return false
				}
			default:
				// For other states, use direct comparison
				if nodeState != state {
					return false
				}
			}
		}
		return true
	}

	// Count how many nodes are in each state for logging
	countNodesInState := func() map[string]int {
		counts := make(map[string]int)
		for _, state := range status.nodes {
			counts[state.String()]++
		}
		return counts
	}

	// Define the state transition sequence based on our expected flow
	// After PUBLISHED, we expect APPLICABLE
	if status.lastGlobalState == grpc_controlplane.UpdateState_UPDATE_PUBLISHED &&
		allNodesInState(grpc_controlplane.UpdateState_UPDATE_APPLICABLE) {

		counts := countNodesInState()
		mqtt_log.Info().
			Str("previous_global_state", status.lastGlobalState.String()).
			Str("new_global_state", "UPDATE_APPLICABLE").
			Interface("node_counts", counts).
			Msg("Global state changing to APPLICABLE - all nodes ready")

		status.lastGlobalState = grpc_controlplane.UpdateState_UPDATE_APPLICABLE
		return grpc_controlplane.UpdateState_UPDATE_APPLICABLE, nil
	}

	// After APPLICABLE and APPLY_REQ, we expect APPLIED
	if (status.lastGlobalState == grpc_controlplane.UpdateState_UPDATE_APPLICABLE ||
		status.lastGlobalState == grpc_controlplane.UpdateState_UPDATE_APPLY_REQ) &&
		allNodesInState(grpc_controlplane.UpdateState_UPDATE_APPLIED) {

		counts := countNodesInState()
		mqtt_log.Info().
			Str("previous_global_state", status.lastGlobalState.String()).
			Str("new_global_state", "UPDATE_APPLIED").
			Interface("node_counts", counts).
			Msg("Global state changing to APPLIED - all nodes have applied update")

		status.lastGlobalState = grpc_controlplane.UpdateState_UPDATE_APPLIED
		return grpc_controlplane.UpdateState_UPDATE_APPLIED, nil
	}

	// Log progress but without changing global state
	counts := countNodesInState()
	totalNodes := len(status.expectedNodes)
	mqtt_log.Debug().
		Str("node", msg.SerialNumber).
		Str("state", msg.State.String()).
		Interface("node_counts", counts).
		Int("total_nodes", totalNodes).
		Msg("Node state updated, waiting for all nodes to reach same state")

	return status.lastGlobalState, nil
}

// Send UPDATE_APPLY_REQ to all nodes
func sendUpdateApplyRequest(
	ctx context.Context,
	c *client,
	req *grpc_controlplane.FleetUpdate,
) error {
	mqtt_log.Info().Msg("Sending apply request to all nodes")

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while sending apply request")
		default:
			// Use the new topic structure for sync control messages
			topicSync := node.SerialNumber + "/control/sync"

			// Create payload with status and transaction ID
			payload := fmt.Sprintf(`{"status": %d, "tx_id": %d}`,
				grpc_controlplane.UpdateState_UPDATE_APPLY_REQ,
				req.Transaction.TxId)

			token := c.client.Publish(topicSync, 2, false, []byte(payload))
			token.Wait()

			if err := token.Error(); err != nil {
				mqtt_log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to send apply request")
				return fmt.Errorf("failed to send apply request to node %s: %w", node.SerialNumber, err)
			}

			mqtt_log.Debug().Str("node", node.SerialNumber).Msg("Apply request sent")
		}
	}

	return nil
}

// Send UPDATE_ACKNOWLEDGED to all nodes
func sendUpdateAcknowledgement(
	ctx context.Context,
	c *client,
	req *grpc_controlplane.FleetUpdate,
) error {
	mqtt_log.Info().Msg("Sending acknowledgement to all nodes")

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while sending acknowledgement")
		default:
			// Use the new topic structure for sync control messages
			topicSync := node.SerialNumber + "/control/sync"

			// Create payload with status and transaction ID
			payload := fmt.Sprintf(`{"status": %d, "tx_id": %d}`,
				grpc_controlplane.UpdateState_UPDATE_ACKNOWLEDGED,
				req.Transaction.TxId)

			token := c.client.Publish(topicSync, 2, false, []byte(payload))
			token.Wait()

			if err := token.Error(); err != nil {
				mqtt_log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to send acknowledgement")
				return fmt.Errorf("failed to send acknowledgement to node %s: %w", node.SerialNumber, err)
			}

			mqtt_log.Debug().Str("node", node.SerialNumber).Msg("Acknowledgement sent")
		}
	}

	return nil
}

// Send UPDATE_ROLLBACK to all nodes
func sendRollbackRequest(
	ctx context.Context,
	c *client,
	req *grpc_controlplane.FleetUpdate,
) error {
	mqtt_log.Info().Msg("Sending rollback request to all nodes")

	for _, node := range req.NodeUpdateItems {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while sending rollback request")
		default:
			// Use the new topic structure for sync control messages
			topicSync := node.SerialNumber + "/control/sync"

			// Create payload with status and transaction ID
			payload := fmt.Sprintf(`{"status": %d, "tx_id": %d}`,
				grpc_controlplane.UpdateState_UPDATE_ROLLBACK,
				req.Transaction.TxId)

			token := c.client.Publish(topicSync, 2, false, []byte(payload))
			token.Wait()

			if err := token.Error(); err != nil {
				mqtt_log.Error().Err(err).Str("node", node.SerialNumber).Msg("Failed to send rollback request")
				// Continue with other nodes even if one fails
				continue
			}

			mqtt_log.Debug().Str("node", node.SerialNumber).Msg("Rollback request sent")
		}
	}

	return nil
}

// Send FleetResponse to client - only for global state changes
func sendFleetResponse(
	stream grpc.ServerStreamingServer[grpc_controlplane.FleetResponse],
	state grpc_controlplane.UpdateState,
	txId int32,
	meta string,
) error {
	return stream.Send(&grpc_controlplane.FleetResponse{
		UpdateState: state,
		Timestamp:   timestamppb.New(time.Now()),
		TxId:        txId,
		Meta:        &meta,
	})
}
