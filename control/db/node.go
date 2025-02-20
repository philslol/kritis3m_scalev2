package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (s *StateManager) CreateNode(ctx context.Context, node *types.Node) (*types.Node, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO nodes (serial_number, network_index, locality, last_seen, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRow(ctx, query,
		node.SerialNumber,
		node.NetworkIndex,
		node.Locality,
		node.LastSeen,
		node.CreatedBy,
	).Scan(
		&node.ID,
		&node.SerialNumber,
		&node.NetworkIndex,
		&node.Locality,
		&node.LastSeen,
		&node.CreatedAt,
		&node.UpdatedAt,
		&node.CreatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert node: %w", err)
	}
	tx.Commit(ctx)

	return node, nil
}

func (s *StateManager) GetNode(ctx context.Context, id int) (*types.Node, error) {
	node := &types.Node{}
	query := `SELECT id, serial_number, network_index, locality, last_seen, created_at, updated_at, created_by 
              FROM nodes WHERE id = $1`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&node.ID,
		&node.SerialNumber,
		&node.NetworkIndex,
		&node.Locality,
		&node.LastSeen,
		&node.CreatedAt,
		&node.UpdatedAt,
		&node.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	return node, nil
}

func (s *StateManager) DeleteNode(ctx context.Context, id int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM nodes WHERE id = $1`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) ListNodes(ctx context.Context, version_set_id *uuid.UUID) ([]*types.Node, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT id, serial_number, network_index, locality, last_seen, created_at, updated_at, created_by, version_set_id, state
		FROM nodes WHERE version_set_id = $1
	`

	// Execute the query
	rows, err := tx.Query(ctx, query, version_set_id.String)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Slice to hold the results
	var nodes []*types.Node

	// Iterate over the rows and populate the nodes slice
	for rows.Next() {
		var node types.Node

		err := rows.Scan(
			&node.ID,
			&node.SerialNumber,
			&node.NetworkIndex,
			&node.Locality,
			node.LastSeen,
			&node.CreatedAt,
			&node.UpdatedAt,
			&node.CreatedBy,
			node.VersionSetID,
			&node.State,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nodes, nil
}
