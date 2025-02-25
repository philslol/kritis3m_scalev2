package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

func (s *StateManager) CreateHwConfig(ctx context.Context, config *types.HardwareConfig) error {

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
        INSERT INTO hardware_configs (
            node_serial, device, ip_cidr, version_set_id, created_by
        ) VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at`

		return s.pool.QueryRow(ctx, query,
			config.NodeSerial, config.Device, config.IPCIDR, config.VersionSetID,
			config.CreatedBy,
		).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
	})
	if err != nil {
		log.Err(err).Msg("failed to create hardware config")
		return err
	}
	return nil
}

func (s *StateManager) GetHwConfigPByID(ctx context.Context, id int) (*types.HardwareConfig, error) {
	config := &types.HardwareConfig{}
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE id = $1`

		return s.pool.QueryRow(ctx, query, id).Scan(
			&config.ID, &config.NodeSerial, &config.Device, &config.IPCIDR,
			&config.VersionSetID, &config.CreatedAt,
			&config.UpdatedAt, &config.CreatedBy,
		)
	})
	if err != nil {
		log.Err(err).Msg("failed to get hardware config by id")
		return nil, err
	}
	return config, nil
}

func (s *StateManager) GetHwConfigbyNodeID(ctx context.Context, nodeID int) ([]*types.HardwareConfig, error) {
	configs := []*types.HardwareConfig{}
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE node_id = $1`

		rows, err := s.pool.Query(ctx, query, nodeID)
		if err != nil {
			return err
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
				return err
			}
			configs = append(configs, config)
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to get hardware config by node id")
		return nil, err
	}
	return configs, nil
}

func (s *StateManager) GetHwConfigBySerial(ctx context.Context, serialNumber string, versionSetID uuid.UUID) ([]*types.HardwareConfig, error) {
	configs := []*types.HardwareConfig{}
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE version_set_id = $1 AND node_serial = $2`

		rows, err := s.pool.Query(ctx, query, versionSetID, serialNumber)
		if err != nil {
			return err
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
				return err
			}
			configs = append(configs, config)
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to get hardware config by serial")
		return nil, err
	}
	return configs, nil
}

func (s *StateManager) GetHwConfigByVersionSetID(ctx context.Context, versionSetID uuid.UUID) ([]*types.HardwareConfig, error) {
	configs := []*types.HardwareConfig{}

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
        SELECT id, node_serial, device, ip_cidr, version_set_id,
               created_at, updated_at, created_by
        FROM hardware_configs WHERE version_set_id = $1`

		rows, err := s.pool.Query(ctx, query, versionSetID)
		if err != nil {
			return err
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
				return err
			}
			configs = append(configs, config)
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to get hardware config by version set id")
		return nil, err
	}
	return configs, nil
}
