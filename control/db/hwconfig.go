package db

import (
	"context"
	"fmt"

	"github.com/philslol/kritis3m_scalev2/control/types"
)

// HardwareConfig CRUD
func (s *StateManager) CreateHardwareConfig(ctx context.Context, config *types.HardwareConfig) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO hardware_configs (
			node_id,  device, ip_cidr, status,
			version, previous_version_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRow(ctx, query,
		config.NodeID, config.Device,
		config.IPCIDR, config.Status, config.Version,
		config.PreviousVersionID, config.CreatedBy,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert hardware config: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) GetHardwareConfig(ctx context.Context, id int) (*types.HardwareConfig, error) {
	config := &types.HardwareConfig{}
	query := `
		SELECT id, node_id, device, ip_cidr,
			   status, version, previous_version_id,
			   created_at, updated_at, created_by
		FROM hardware_configs WHERE id = $1`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.NodeID,
		&config.Device, &config.IPCIDR, &config.Status,
		&config.Version, &config.PreviousVersionID,
		&config.CreatedAt, &config.UpdatedAt, &config.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get hardware config: %w", err)
	}
	return config, nil
}

func (s *StateManager) UpdateHardwareConfig(ctx context.Context, config *types.HardwareConfig) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE hardware_configs
		SET device = $1, ip_cidr = $2, status = $3,
			version = $4, previous_version_id = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at`

	err = tx.QueryRow(ctx, query,
		config.Device, config.IPCIDR, config.Status,
		config.Version, config.PreviousVersionID, config.ID,
	).Scan(&config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update hardware config: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) DeleteHardwareConfig(ctx context.Context, id int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM hardware_configs WHERE id = $1`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete hardware config: %w", err)
	}

	return tx.Commit(ctx)
}
