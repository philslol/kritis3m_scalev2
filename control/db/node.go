package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Node represents a node in the system

// Node represents a node in the system
type Node struct {
	ID           int                `json:"id"`
	SerialNumber string             `json:"serial_number"`
	NetworkIndex int                `json:"network_index"`
	Locality     string             `json:"locality"`
	LastSeen     pgtype.Timestamptz `json:"last_seen"`
	ValidFrom    time.Time          `json:"valid_from"`
	ValidTo      pgtype.Timestamptz `json:"valid_to"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	CreatedBy    string             `json:"created_by"`
}

// CRUD Functions

func (s *StateManager) CreateNode(ctx context.Context, transactionID uuid.UUID, node *Node) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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
	).Scan(&node.ID, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert node: %w", err)
	}

	// Log the change
	newData, _ := json.Marshal(node)
	err = s.LogChange(ctx, tx, transactionID, "nodes", fmt.Sprintf("%d", node.ID), "INSERT", "", string(newData), node.CreatedBy)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *StateManager) GetNode(ctx context.Context, id int) (*Node, error) {
	node := &Node{}
	query := `SELECT id, serial_number, network_index, locality, last_seen, valid_from, valid_to, created_at, updated_at, created_by 
              FROM nodes WHERE id = $1`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&node.ID,
		&node.SerialNumber,
		&node.NetworkIndex,
		&node.Locality,
		&node.LastSeen,
		&node.ValidFrom,
		&node.ValidTo,
		&node.CreatedAt,
		&node.UpdatedAt,
		&node.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	return node, nil
}

func (s *StateManager) UpdateNode(ctx context.Context, transactionID uuid.UUID, node *Node) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get old data for change log
	oldNode, err := s.GetNode(ctx, node.ID)
	if err != nil {
		return err
	}
	oldData, _ := json.Marshal(oldNode)

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
	).Scan(&node.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	// Log the change
	newData, _ := json.Marshal(node)
	err = s.LogChange(ctx, tx, transactionID, "nodes", fmt.Sprintf("%d", node.ID), "UPDATE", string(oldData), string(newData), node.CreatedBy)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *StateManager) DeleteNode(ctx context.Context, transactionID uuid.UUID, id int, createdBy string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get old data for change log
	oldNode, err := s.GetNode(ctx, id)
	if err != nil {
		return err
	}
	oldData, _ := json.Marshal(oldNode)

	query := `DELETE FROM nodes WHERE id = $1`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	// Log the change
	err = s.LogChange(ctx, tx, transactionID, "nodes", fmt.Sprintf("%d", id), "DELETE", string(oldData), "", createdBy)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
