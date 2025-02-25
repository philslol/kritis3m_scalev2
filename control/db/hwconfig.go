package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (s *StateManager) CreateHwConfig(ctx context.Context, config *types.HardwareConfig) error {
	query := `
        INSERT INTO hardware_configs (
            node_serial, device, ip_cidr, version_set_id, created_by
        ) VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at`

	return s.pool.QueryRow(ctx, query,
		config.NodeSerial, config.Device, config.IPCIDR, config.VersionSetID,
		config.CreatedBy,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
}

func (s *StateManager) GetHwConfigPByID(ctx context.Context, id int) (*types.HardwareConfig, error) {
	config := &types.HardwareConfig{}
	query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE id = $1`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.NodeSerial, &config.Device, &config.IPCIDR,
		&config.VersionSetID, &config.CreatedAt,
		&config.UpdatedAt, &config.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (s *StateManager) GetHwConfigbyNodeID(ctx context.Context, nodeID int) ([]*types.HardwareConfig, error) {
	configs := []*types.HardwareConfig{}
	query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE node_id = $1`

	rows, err := s.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		config := &types.HardwareConfig{}
		err := rows.Scan(
			&config.ID, &config.NodeSerial, &config.Device, &config.IPCIDR,
			&config.VersionSetID, &config.CreatedAt,
			&config.UpdatedAt, &config.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func (r *StateManager) GetHwConfigBySerial(ctx context.Context, serialNumber string, versionSetID uuid.UUID) ([]*types.HardwareConfig, error) {
	configs := []*types.HardwareConfig{}
	query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE version_set_id = $1 AND node_serial = $2`

	rows, err := r.pool.Query(ctx, query, versionSetID, serialNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		config := &types.HardwareConfig{}
		err := rows.Scan(
			&config.ID, &config.NodeSerial, &config.Device, &config.IPCIDR,
			&config.VersionSetID, &config.CreatedAt,
			&config.UpdatedAt, &config.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func (r *StateManager) GetHwConfigByVersionSetID(ctx context.Context, versionSetID uuid.UUID) ([]*types.HardwareConfig, error) {
	configs := []*types.HardwareConfig{}
	query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE version_set_id = $1`

	rows, err := r.pool.Query(ctx, query, versionSetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		config := &types.HardwareConfig{}
		err := rows.Scan(
			&config.ID, &config.NodeSerial, &config.Device, &config.IPCIDR,
			&config.VersionSetID, &config.CreatedAt,
			&config.UpdatedAt, &config.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}
