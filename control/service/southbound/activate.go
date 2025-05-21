package southbound

import (
	"context"
	"fmt"
	"io"
	"time"

	grpc_controlplane "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/control_plane"
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func getControlPlaneClient(addr string) (grpc_controlplane.ControlPlaneClient, *grpc.ClientConn, error) {
	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	conn, err := grpc.NewClient(addr, grpcOptions...)
	if err != nil {
		log.Error().Err(err).Msg("Could not connect to control plane")
		return nil, nil, status.Error(codes.Internal, "Failed to connect to control plane")
	}

	client := grpc_controlplane.NewControlPlaneClient(conn)
	return client, conn, nil
}

func (sb *SouthboundService) ActivateFleet(ctx context.Context, req *grpc_southbound.ActivateFleetRequest) (*grpc_southbound.ActivateResponse, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute) // Longer timeout for fleet updates
	defer cancel()

	// Validate request
	if req.VersionSetId == "" {
		return nil, status.Error(codes.InvalidArgument, "VersionSetId is required")
	}

	// Parse version set UUID
	uuid_version_set, err := uuid.FromString(req.VersionSetId)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse version set id")
		return nil, status.Error(codes.InvalidArgument, "Invalid VersionSetId format")
	}

	// Determine update type and get fleet update
	var fleetUpdate *grpc_controlplane.FleetUpdate
	var description string
	var transactionType types.TransactionType
	var version_transition_id int
	var fromVersionTransition *int = nil

	if req.GroupName != nil && *req.GroupName != "" {
		// This is a group update
		description = fmt.Sprintf("Group Update for %s in VersionSet %s", *req.GroupName, req.VersionSetId)
		transactionType = types.TransactionTypeGroupUpdate
		fleetUpdate, err = sb.db.GetFleetUpdateOptimized(ctx, req.VersionSetId, *req.GroupName)
	} else {
		// This is a version update
		description = fmt.Sprintf("Version Update to %s", req.VersionSetId)
		transactionType = types.TransactionTypeVersionUpdate
		fleetUpdate, err = sb.db.GetFleetUpdateOptimized(ctx, req.VersionSetId, "")
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to get fleet update")
		return nil, status.Error(codes.Internal, "Failed to get fleet update")
	}

	if fleetUpdate == nil || len(fleetUpdate.NodeUpdateItems) == 0 {
		return nil, status.Error(codes.NotFound, "No nodes found for update")
	}

	// Create transaction
	tx, err := sb.db.CreateTransaction(ctx, description, transactionType)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create transaction")
		return nil, status.Error(codes.Internal, "Failed to create transaction")
	}

	// If this is a version update, create a version transition
	if transactionType == types.TransactionTypeVersionUpdate {
		transition := &types.VersionTransition{
			ToVersionSetID: uuid_version_set,
			Status:         "pending",
			CreatedBy:      "system",
			TransactionID:  int(tx),
			StartedAt:      time.Now(),
		}
		var last_version_transition_id int
		err := sb.db.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
			query := `
			SELECT id FROM version_transitions
			WHERE status = 'active' and disabled_at is NULL
			LIMIT 1
			`
			rows, err := tx.Query(ctx, query)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				err = rows.Scan(&last_version_transition_id)
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			log.Warn().Err(err).Msg("Failed to get last version transition id")
			fromVersionTransition = nil
		} else if last_version_transition_id != 0 {
			fromVersionTransition = &last_version_transition_id
		} else {
			fromVersionTransition = nil
		}
		transition.FromVersionTransition = fromVersionTransition

		version_transition_id, err = sb.db.CreateVersionTransition(ctx, transition)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create version transition")
			return nil, status.Error(codes.Internal, "Failed to create version transition")
		}

	}

	// Get client with error handling
	client, conn, err := getControlPlaneClient(sb.addr)
	if err != nil {
		if logErr := sb.logTransactionFailure(ctx, tx, "fleet", uuid_version_set, "Failed to connect to control plane"); logErr != nil {
			log.Error().Err(logErr).Msg("Failed to log transaction failure")
		}
		return nil, err
	}
	defer conn.Close()

	// Set transaction ID in fleet update
	fleetUpdate.Transaction = &grpc_controlplane.Transaction{
		TxId: int32(tx),
	}

	// Start update stream
	stream, err := client.UpdateFleet(ctx, fleetUpdate)
	if err != nil {
		if logErr := sb.logTransactionFailure(ctx, tx, "fleet", uuid_version_set, fmt.Sprintf("Failed to start fleet update: %v", err)); logErr != nil {
			log.Error().Err(logErr).Msg("Failed to log transaction failure")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to start fleet update: %v", err))
	}

	var retcode int32
	var lastError error
	var finalState types.TransactionState
	var finalDescription string

	// Create done channel for graceful shutdown
	done := make(chan struct{})
	defer close(done)

	// Start goroutine to handle stream receiving
	go func() {

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled, set error state and return
				finalState = types.TransactionStateError
				finalDescription = fmt.Sprintf("Operation cancelled: %v", ctx.Err())
				return
			case <-done:
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						// Connection closed by peer, mark as success if no errors occurred
						if finalState == "" {
							finalState = types.TransactionStateApplied
							finalDescription = "Update completed successfully"
						}
						done <- struct{}{}
						return
					}
					lastError = err
					log.Error().Err(err).Msg("Failed to receive fleet update response")
					finalState = types.TransactionStateError
					finalDescription = fmt.Sprintf("Stream error: %v", err)
					return
				}

				// Map update state to transaction state
				var state types.TransactionState
				switch resp.UpdateState {
				case grpc_controlplane.UpdateState_UPDATE_APPLIED:
					state = types.TransactionStateApplied
				case grpc_controlplane.UpdateState_UPDATE_ERROR:
					state = types.TransactionStateError
				case grpc_controlplane.UpdateState_UPDATE_APPLY_REQ:
					state = types.TransactionStateApplicable
				case grpc_controlplane.UpdateState_UPDATE_APPLICABLE:
					state = types.TransactionStateApplicable
				case grpc_controlplane.UpdateState_UPDATE_PUBLISHED:
					state = types.TransactionStatePublished
				}

				// Log node transaction
				_, err = sb.db.LogNodeTransaction(ctx, &types.NodeTransactionLog{
					TransactionID: int(tx),
					NodeSerial:    resp.SerialNumber,
					VersionSetID:  uuid_version_set,
					State:         state,
					Timestamp:     resp.Timestamp.AsTime(),
				})
				if err != nil {
					log.Error().Err(err).Msg("Failed to log node transaction")
				}

				// Handle error state
				if state == types.TransactionStateError {
					finalState = types.TransactionStateError
					finalDescription = fmt.Sprintf("Node %s reported error: %s", resp.SerialNumber, *resp.Meta)
					return
				}
			}
		}
	}()

	// Wait for completion or timeout
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			if logErr := sb.logTransactionFailure(ctx, tx, "fleet", uuid_version_set, "Operation timed out"); logErr != nil {
				log.Error().Err(logErr).Msg("Failed to log timeout")
			}
			// get update type fleet or group
			if transactionType == types.TransactionTypeVersionUpdate {
				err = sb.db.UpdateVersionTransitionStatus(ctx, version_transition_id, "failed", nil)
				if err != nil {
					log.Error().Err(err).Msg("Failed to update version transition status")
				}
			}

			return nil, status.Error(codes.DeadlineExceeded, "Operation timed out")
		}
		return nil, ctx.Err()
	case <-done:
		if lastError != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Operation failed: %v", lastError))
		}

		// Update transaction with final state
		completed_at := time.Now()
		err = sb.db.UpdateTransaction(ctx, int(tx), &completed_at, &finalState, &finalDescription)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update transaction")
		}

		// Update version transition if needed
		if transactionType == types.TransactionTypeVersionUpdate {
			status := "failed"
			if finalState == types.TransactionStateApplied {
				status = "active"

				// revoke old version transition
				disabled_at := time.Now()
				if fromVersionTransition != nil {
					err = sb.db.UpdateVersionTransitionStatus(ctx, *fromVersionTransition, "disabled", &disabled_at)
					if err != nil {
						log.Error().Err(err).Msg("Failed to revoke old version transition")
					}
				}
			}
			err = sb.db.UpdateVersionTransitionStatus(ctx, version_transition_id, status, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to update version transition status")
			}
		}

		// Set return code based on final state
		if finalState == types.TransactionStateError {
			retcode = -1
		} else {
			retcode = 0
		}

		return &grpc_southbound.ActivateResponse{
			Retcode: retcode,
		}, nil
	}
}

func (sb *SouthboundService) ActivateNode(ctx context.Context, req *grpc_southbound.ActivateNodeRequest) (*grpc_southbound.ActivateResponse, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute) // Longer timeout for node updates
	defer cancel()

	// Check arguments
	if req.SerialNumber == "" || req.VersionSetId == "" {
		return nil, status.Error(codes.InvalidArgument, "SerialNumber and VersionSetId are required")
	}

	// Get node update from database
	nodeUpdate, err := sb.db.NodeUpdate(req.SerialNumber, req.VersionSetId, ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get node update from database")
		return nil, status.Error(codes.Internal, "Failed to get node update")
	}
	if nodeUpdate == nil {
		return nil, status.Error(codes.NotFound, "Node not found or no updates available")
	}

	// Create transaction first to ensure we have a valid transaction ID
	description := fmt.Sprintf("Activate Node %s", req.SerialNumber)
	tx, err := sb.db.CreateTransaction(ctx, description, types.TransactionTypeNodeUpdate)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create transaction")
		return nil, status.Error(codes.Internal, "Failed to create transaction")
	}

	// Parse UUID once
	uuid_version_set, err := uuid.FromString(req.VersionSetId)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse version set id")
		return nil, status.Error(codes.Internal, "Failed to parse version set id")
	}

	// Get client with error handling
	client, conn, err := getControlPlaneClient(sb.addr)
	if err != nil {
		// Log transaction failure before returning
		if logErr := sb.logTransactionFailure(ctx, tx, req.SerialNumber, uuid_version_set, "Failed to connect to control plane"); logErr != nil {
			log.Error().Err(logErr).Msg("Failed to log transaction failure")
		}
		return nil, err
	}
	defer conn.Close()

	// Create the node update request with proper transaction ID type
	update := &grpc_controlplane.NodeUpdate{
		NodeUpdateItem: nodeUpdate,
		Transaction: &grpc_controlplane.Transaction{
			TxId: int32(tx),
		},
	}

	// Start update stream with timeout context
	stream, err := client.UpdateNode(ctx, update)
	if err != nil {
		if logErr := sb.logTransactionFailure(ctx, tx, req.SerialNumber, uuid_version_set, fmt.Sprintf("Failed to start update: %v", err)); logErr != nil {
			log.Error().Err(logErr).Msg("Failed to log transaction failure")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to start node update: %v", err))
	}

	var retcode int32
	var lastError error

	// Create done channel for graceful shutdown
	done := make(chan error)
	defer close(done)

	// Start goroutine to handle stream receiving
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			default:
				stream_resp, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						return
					}
					lastError = err
					log.Err(err).Msg("Failed to receive response")
					return
				}

				// Log transaction
				var state types.TransactionState
				if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_APPLIED {
					state = types.TransactionStateApplied
				} else if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_ERROR {
					state = types.TransactionStateError
				} else if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_APPLY_REQ {
					state = types.TransactionStateApplicable
				} else if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_APPLICABLE {
					state = types.TransactionStateApplicable
				} else if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_PUBLISHED {
					state = types.TransactionStatePublished
				}

				_, err = sb.db.LogNodeTransaction(ctx, &types.NodeTransactionLog{
					TransactionID: int(tx),
					NodeSerial:    req.SerialNumber,
					VersionSetID:  uuid_version_set,
					State:         state,
					Timestamp:     stream_resp.Timestamp.AsTime(),
				})
				if err != nil {
					lastError = err
					log.Error().Err(err).Msg("Failed to log transaction")
					return
				}

				// Handle terminal states
				if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_ERROR || stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_APPLIED {
					if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_ERROR {
						retcode = -1
					} else if stream_resp.UpdateState == grpc_controlplane.UpdateState_UPDATE_APPLIED {
						retcode = 0
					}

					// Update transaction completion
					where := fmt.Sprintf("id = %d", tx)
					err = sb.db.UpdateWhere(ctx, "transactions", map[string]any{
						"completed_at": time.Now(),
					}, where)

					if err != nil {
						lastError = err
						log.Error().Err(err).Msg("Failed to set transaction completed")
					}
					done <- nil
					return
				}
			}
		}
	}()

	// Wait for completion or timeout
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			if logErr := sb.logTransactionFailure(ctx, tx, req.SerialNumber, uuid_version_set, "Operation timed out"); logErr != nil {
				log.Error().Err(logErr).Msg("Failed to log timeout")
			}
			return nil, status.Error(codes.DeadlineExceeded, "Operation timed out")
		}
		return nil, ctx.Err()
	case <-done:
		if lastError != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Operation failed: %v", lastError))
		}
		return &grpc_southbound.ActivateResponse{
			Retcode: retcode,
		}, nil
	}
}

// Helper function to log transaction failures
func (sb *SouthboundService) logTransactionFailure(ctx context.Context, tx int, serialNumber string, versionSetID uuid.UUID, errorMsg string) error {
	_, err := sb.db.LogNodeTransaction(ctx, &types.NodeTransactionLog{
		TransactionID: tx,
		NodeSerial:    serialNumber,
		VersionSetID:  versionSetID,
		State:         types.TransactionState(grpc_controlplane.UpdateState_UPDATE_ERROR),
	})
	return err
}
