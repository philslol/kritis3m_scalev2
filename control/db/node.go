package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

func (s *StateManager) CreateNode(ctx context.Context, node *types.Node) (*types.Node, error) {
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		INSERT INTO nodes (serial_number, network_index, locality, version_set_id, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

		return tx.QueryRow(ctx, query,
			node.SerialNumber,
			node.NetworkIndex,
			node.Locality,
			node.VersionSetID,
			node.CreatedBy,
		).Scan(&node.ID, &node.CreatedAt, &node.UpdatedAt)
	})
	if err != nil {
		log.Err(err).Msg("failed to create node")
		return nil, err
	}

	return node, nil
}

func (s *StateManager) GetNodebyID(ctx context.Context, Id int) (*types.Node, error) {
	node := &types.Node{}
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT id, serial_number, network_index, locality, last_seen, version_set_id, 
		       created_at, updated_at, created_by
		FROM nodes 
		WHERE id = $1`

		err := tx.QueryRow(ctx, query, Id).Scan(
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
			return err
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to get node by id")
		return nil, err
	}

	return node, nil
}

func (s *StateManager) ListNodes(ctx context.Context, versionSetID *uuid.UUID) ([]*types.Node, error) {
	var nodes []*types.Node

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {

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

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

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
				return err
			}
			nodes = append(nodes, node)
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to list nodes")
		return nil, err
	}

	return nodes, nil
}

func (s *StateManager) DeleteNode(ctx context.Context, serialNumber string, versionSetID uuid.UUID) error {
	return s.Delete(ctx, "nodes", "serial_number", serialNumber)
}

func (s *StateManager) GetNodebySerial(ctx context.Context, serialNumber string, versionSetID uuid.UUID) (*types.Node, error) {
	node := &types.Node{}
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {

		query := `
		SELECT id, serial_number, network_index, locality, last_seen, version_set_id, 
		       created_at, updated_at, created_by
		FROM nodes 
		WHERE serial_number = $1 AND version_set_id = $2`

		return tx.QueryRow(ctx, query, serialNumber, versionSetID).Scan(
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
	})

	if err != nil {
		log.Err(err).Msg("failed to get node by serial")
		return nil, err
	}

	return node, nil
}
