package db

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/gofrs/uuid/v5"
	// uuidpq "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
)

// completeTransaction completes the current pending transaction.
func (sm *StateManager) CompleteTransaction(ctx context.Context) error {
	tx, err := sm.pool.Begin(ctx)
	_, err = tx.Exec(ctx, `SELECT complete_transaction()`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// rollbackTransaction rolls back the current pending transaction.
func (sm *StateManager) rollbackTransaction(ctx context.Context) error {
	tx, err := sm.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `SELECT rollback_transaction()`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

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

func (s *StateManager) ApplyChanges(ctx context.Context, transactionID uuid.UUID, createdBy string, apply_state bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) // Ensure rollback if failure occurs

	// Simulated logic: attempt network updates
	if !apply_state {
		log.Debug().Msg("Network update failed, rolling back")
		err = s.rollbackTransaction(ctx)
		return fmt.Errorf("network update failed")
	}

	// Mark transaction as successful
	_, err = tx.Exec(ctx, `UPDATE transactions SET status = 'active', completed_at = NOW() WHERE id = $1`, transactionID)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %v", err)
	}

	return tx.Commit(ctx)
}

// ExecuteInTransaction executes the given operation within a transaction
func (s *StateManager) ExecuteInTransaction(ctx context.Context, operation func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Will be ignored if transaction is committed

	if err := operation(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
