package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/gofrs/uuid/v5"
	// uuidpq "github.com/jackc/pgx-gofrs-uuid"
	"golang.org/x/exp/rand"
)

// StartTransaction initializes a new transaction and returns the transaction ID.
func (s *StateManager) StartTransaction(ctx context.Context, createdBy, description string) (uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx) // Rollback if an error occurred before commit
		}
	}()

	query := `INSERT INTO transactions (status, created_at, created_by, description, metadata) 
          VALUES ('pending', NOW(), $1, $2, '{}'::jsonb) RETURNING id`

	var transactionID uuid.UUID
	err = tx.QueryRow(ctx, query, createdBy, description).Scan(&transactionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert transaction: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return transactionID, nil
}


// LogChange records a change in the database within an active transaction.
func (s *StateManager) LogChange(ctx context.Context, tx pgx.Tx, transactionID uuid.UUID, tableName, recordID, operation string, oldData, newData interface{}, createdBy string) error {
	var oldDataJSON, newDataJSON []byte
	var err error

	// Convert oldData to JSON if not nil
	if oldData != nil {
		oldDataJSON, err = json.Marshal(oldData)
		if err != nil {
			return fmt.Errorf("failed to marshal old data: %w", err)
		}
	}

	// Convert newData to JSON if not nil
	if newData != nil {
		newDataJSON, err = json.Marshal(newData)
		if err != nil {
			return fmt.Errorf("failed to marshal new data: %w", err)
		}
	}

	query := `INSERT INTO change_log (id, transaction_id, table_name, record_id, operation, old_data, new_data, created_at, created_by) 
          VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)`

	_, err = tx.Exec(ctx, query,
		uuid.Must(uuid.NewV4()),
		transactionID,
		tableName,
		recordID,
		operation,
		oldDataJSON,
		newDataJSON,
		createdBy,
	)
	if err != nil {
		return fmt.Errorf("failed to log change: %w", err)
	}
	return nil
}

func (s *StateManager) ApplyChanges(ctx context.Context, transactionID uuid.UUID, createdBy string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) // Ensure rollback if failure occurs

	// Simulated logic: attempt network updates
	if !ApplyNetworkChanges(transactionID) {
		log.Debug().Msg("Network update failed, rolling back")
		rollbackTransaction(ctx, tx, transactionID)
		return fmt.Errorf("network update failed")
	}

	// Mark transaction as successful
	_, err = tx.Exec(ctx, `UPDATE transactions SET status = 'active', completed_at = NOW() WHERE id = $1`, transactionID)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %v", err)
	}

	return tx.Commit(ctx)
}

func ApplyNetworkChanges(transactionID uuid.UUID) bool {
	// Initialize the random number generator
	rand.Seed(rand.Uint64())
	// Generate a random number between 0 and 99
	if rand.Intn(100) < 30 { // 30% chance to fail
		fmt.Printf("Network changes for transaction %v failed.\n", transactionID)
		return false // Simulate failure
	}
	// If the random number is 30 or greater, success
	fmt.Printf("Network changes for transaction %v applied successfully.\n", transactionID)
	return true
}

func rollbackTransaction(ctx context.Context, tx pgx.Tx, transactionID uuid.UUID) {
	_, err := tx.Exec(ctx, `UPDATE transactions SET status = 'rollback', completed_at = NOW() WHERE id = $1`, transactionID)
	if err != nil {
		log.Printf("Failed to mark transaction as rollback: %v", err)
	}
}
