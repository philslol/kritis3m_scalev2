package db

import (
	"context"
	"fmt"

	"github.com/philslol/kritis3m_scalev2/control/types"
)

// Node represents a node in the system

// Node represents a node in the system

// CRUD Functions

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

func (s *StateManager) UpdateNode(ctx context.Context, node *types.Node) (*types.Node, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	updatedNode := &types.Node{}

	query := `
		UPDATE nodes 
		SET serial_number = $1, network_index = $2, locality = $3, last_seen = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at`

	err = tx.QueryRow(ctx, query,
		node.SerialNumber,
		node.NetworkIndex,
		node.Locality,
		node.LastSeen,
		node.ID,
	).Scan(
		&updatedNode.ID,
		&updatedNode.SerialNumber,
		&updatedNode.NetworkIndex,
		&updatedNode.Locality,
		&updatedNode.LastSeen,
		&updatedNode.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return updatedNode, nil
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
