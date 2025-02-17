package db

import (
	"context"
	"fmt"

	"github.com/philslol/kritis3m_scalev2/control/types"
)

// Group CRUD
func (s *StateManager) CreateGroup(ctx context.Context, group *types.Group) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO groups (
			 name, log_level, endpoint_config_id,
			legacy_config_id, status, version, previous_version_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRow(ctx, query,
		group.Name, group.LogLevel,
		group.EndpointConfigID, group.LegacyConfigID, group.Status,
		group.Version, group.PreviousVersionID, group.CreatedBy,
	).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert group: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) GetGroup(ctx context.Context, id int) (*types.Group, error) {
	group := &types.Group{}
	query := `
		SELECT id, name, log_level, endpoint_config_id,
			   legacy_config_id, status, version, previous_version_id,
			   created_at, updated_at, created_by
		FROM groups WHERE id = $1`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&group.ID,  &group.Name, &group.LogLevel,
		&group.EndpointConfigID, &group.LegacyConfigID, &group.Status,
		&group.Version, &group.PreviousVersionID, &group.CreatedAt,
		&group.UpdatedAt, &group.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return group, nil
}

func (s *StateManager) UpdateGroup(ctx context.Context, group *types.Group) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE groups
		SET name = $1, log_level = $2, endpoint_config_id = $3,
			legacy_config_id = $4, status = $5, version = $6,
			previous_version_id = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at`

	err = tx.QueryRow(ctx, query,
		group.Name, group.LogLevel, group.EndpointConfigID,
		group.LegacyConfigID, group.Status, group.Version,
		group.PreviousVersionID, group.ID,
	).Scan(&group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) DeleteGroup(ctx context.Context, id int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM groups WHERE id = $1`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return tx.Commit(ctx)
}
