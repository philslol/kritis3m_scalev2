package southbound

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func getControlPlaneClient(addr string) (v1.ControlPlaneClient, *grpc.ClientConn, error) {
	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	conn, err := grpc.NewClient(addr, grpcOptions...)
	if err != nil {
		log.Error().Err(err).Msg("Could not connect to control plane")
		return nil, nil, status.Error(codes.Internal, "Failed to connect to control plane")
	}

	client := v1.NewControlPlaneClient(conn)
	return client, conn, nil
}

func (sb *SouthboundService) ActivateFleet(ctx context.Context, req *v1.ActivateFleetRequest) (*v1.ActivateResponse, error) {
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
	var fleetUpdate *v1.FleetUpdate
	var description string
	var transactionType types.TransactionType

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
		}

		err = sb.db.CreateVersionTransition(ctx, transition)
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
	fleetUpdate.Transaction = &v1.Transaction{
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

	// Create done channel for graceful shutdown
	done := make(chan struct{})
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
				resp, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						return
					}
					lastError = err
					log.Error().Err(err).Msg("Failed to receive fleet update response")
					return
				}

				// Process global fleet state update
				log.Info().
					Str("state", resp.UpdateState.String()).
					Int32("tx_id", resp.TxId).
					Str("meta", resp.Meta).
					Msg("Fleet update state changed")

				// Log transaction for each node with the current global state
				for _, node := range fleetUpdate.NodeUpdateItems {
					_, err = sb.db.LogNodeTransaction(ctx, &types.NodeTransactionLog{
						TransactionID: int(tx),
						NodeSerial:    node.SerialNumber,
						VersionSetID:  uuid_version_set,
						State:         types.TransactionState(resp.UpdateState),
					})

					if err != nil {
						lastError = err
						log.Error().Err(err).Msg("Failed to log transaction")
						return
					}
				}

				// Handle terminal states
				if resp.UpdateState == v1.UpdateState_UPDATE_ERROR || resp.UpdateState == v1.UpdateState_UPDATE_APPLIED {
					if resp.UpdateState == v1.UpdateState_UPDATE_ERROR {
						completed_at := time.Now()
						error_state := types.TransactionStateError
						error_description := fmt.Sprintf("Failed to apply update: %s", resp.Meta)
						sb.db.UpdateTransaction(ctx, int(tx), &completed_at, &error_state, &error_description)
						retcode = -1
						// Update version transition status if this was a version update
						if transactionType == types.TransactionTypeVersionUpdate {
							err = sb.db.UpdateVersionTransitionStatus(ctx, int(tx), "failed")
							if err != nil {
								log.Error().Err(err).Msg("Failed to update version transition status")
							}
						}
					} else if resp.UpdateState == v1.UpdateState_UPDATE_APPLIED {
						completed_at := time.Now()
						applied_state := types.TransactionStateApplied
						applied_description := "Update applied successfully"
						sb.db.UpdateTransaction(ctx, int(tx), &completed_at, &applied_state, &applied_description)
						retcode = 0
						// Update version transition status if this was a version update
						if transactionType == types.TransactionTypeVersionUpdate {
							err = sb.db.UpdateVersionTransitionStatus(ctx, int(tx), "active")
							if err != nil {
								log.Error().Err(err).Msg("Failed to update version transition status")
							}
						}
					}

					// Mark transaction as completed
					where := fmt.Sprintf("id = %d", tx)
					err = sb.db.UpdateWhere(ctx, "transactions", map[string]any{
						"completed_at": time.Now(),
					}, where)

					if err != nil {
						lastError = err
						log.Error().Err(err).Msg("Failed to set transaction completed")
						return
					}
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
			return nil, status.Error(codes.DeadlineExceeded, "Operation timed out")
		}
		return nil, ctx.Err()
	case <-done:
		if lastError != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Operation failed: %v", lastError))
		}
		return &v1.ActivateResponse{
			Retcode: retcode,
		}, nil
	}
}

func (sb *SouthboundService) ActivateNode(ctx context.Context, req *v1.ActivateNodeRequest) (*v1.ActivateResponse, error) {
	// Add timeout to context

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
	update := &v1.NodeUpdate{
		NodeUpdateItem: nodeUpdate,
		Transaction: &v1.Transaction{
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
				if stream_resp.UpdateState == v1.UpdateState_UPDATE_APPLIED {
					state = types.TransactionStateApplied
				} else if stream_resp.UpdateState == v1.UpdateState_UPDATE_ERROR {
					state = types.TransactionStateError
				} else if stream_resp.UpdateState == v1.UpdateState_UPDATE_APPLY_REQ {
					state = types.TransactionStateApplicable
				} else if stream_resp.UpdateState == v1.UpdateState_UPDATE_APPLICABLE {
					state = types.TransactionStateApplicable
				} else if stream_resp.UpdateState == v1.UpdateState_UPDATE_PUBLISHED {
					state = types.TransactionStatePublished
				}

				_, err = sb.db.LogNodeTransaction(ctx, &types.NodeTransactionLog{
					TransactionID: int(tx),
					NodeSerial:    req.SerialNumber,
					VersionSetID:  uuid_version_set,
					State:         state,
				})
				if err != nil {
					lastError = err
					log.Error().Err(err).Msg("Failed to log transaction")
					return
				}

				// Handle terminal states
				if stream_resp.UpdateState == v1.UpdateState_UPDATE_ERROR || stream_resp.UpdateState == v1.UpdateState_UPDATE_APPLIED {
					if stream_resp.UpdateState == v1.UpdateState_UPDATE_ERROR {
						retcode = -1
					} else if stream_resp.UpdateState == v1.UpdateState_UPDATE_APPLIED {
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
		return &v1.ActivateResponse{
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
		State:         types.TransactionState(v1.UpdateState_UPDATE_ERROR),
	})
	return err
}
