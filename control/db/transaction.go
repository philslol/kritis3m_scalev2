package db

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
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
