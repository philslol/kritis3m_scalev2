package db

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (s *StateManager) CreateNode(ctx context.Context, node *types.Node) (*types.Node, error) {
	query := `
		INSERT INTO nodes (serial_number, network_index, locality, version_set_id, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := s.pool.QueryRow(ctx, query,
		node.SerialNumber,
		node.NetworkIndex,
		node.Locality,
		node.VersionSetID,
		node.CreatedBy,
	).Scan(&node.ID, &node.CreatedAt, &node.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return node, nil
}

func (s *StateManager) GetNodebyID(ctx context.Context, Id int) (*types.Node, error) {
	query := `
		SELECT id, serial_number, network_index, locality, last_seen, version_set_id, 
		       created_at, updated_at, created_by
		FROM nodes 
		WHERE id = $1`

	node := &types.Node{}
	err := s.pool.QueryRow(ctx, query, Id).Scan(
		&node.ID,
		&node.SerialNumber,
		&node.NetworkIndex,
		&node.Locality,
		&node.LastSeen,
		&node.VersionSetID,
		&node.CreatedAt,
		&node.UpdatedAt,
		&node.CreatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}

func (s *StateManager) ListNodes(ctx context.Context, versionSetID *uuid.UUID) ([]*types.Node, error) {
	var query string
	var args []interface{}

	if versionSetID != nil {
		query = `
			SELECT id, serial_number, network_index, locality, last_seen, version_set_id,
			       created_at, updated_at, created_by
			FROM nodes 
			WHERE version_set_id = $1`
		args = []interface{}{versionSetID}
	} else {
		query = `
			SELECT id, serial_number, network_index, locality, last_seen, version_set_id,
			       created_at, updated_at, created_by
			FROM nodes`
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*types.Node
	for rows.Next() {
		node := &types.Node{}
		err := rows.Scan(
			&node.ID,
			&node.SerialNumber,
			&node.NetworkIndex,
			&node.Locality,
			&node.LastSeen,
			&node.VersionSetID,
			&node.CreatedAt,
			&node.UpdatedAt,
			&node.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (s *StateManager) DeleteNode(ctx context.Context, serialNumber string, versionSetID uuid.UUID) error {
	query := `DELETE FROM nodes WHERE serial_number = $1 AND version_set_id = $2`

	_, err := s.pool.Exec(ctx, query, serialNumber, versionSetID)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

func (s *StateManager) GetNodebySerial(ctx context.Context, serialNumber string, versionSetID uuid.UUID) (*types.Node, error) {
	query := `
		SELECT id, serial_number, network_index, locality, last_seen, version_set_id, 
		       created_at, updated_at, created_by
		FROM nodes 
		WHERE serial_number = $1 AND version_set_id = $2`

	node := &types.Node{}
	err := s.pool.QueryRow(ctx, query, serialNumber, versionSetID).Scan(
		&node.ID,
		&node.SerialNumber,
		&node.NetworkIndex,
		&node.Locality,
		&node.LastSeen,
		&node.VersionSetID,
		&node.CreatedAt,
		&node.UpdatedAt,
		&node.CreatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return node, nil
}
