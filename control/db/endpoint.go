package db

import (
	"context"
	"fmt"

	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

func (s *StateManager) CreateEndpointConfig(ctx context.Context, config *types.EndpointConfig) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Err(err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO endpoint_configs (name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRow(ctx, query,
		config.Name,
		config.MutualAuth,
		config.NoEncryption,
		config.ASLKeyExchangeMethod,
		config.Cipher,
		config.VersionSetID,
		config.CreatedBy,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		log.Err(err)
		return fmt.Errorf("failed to insert endpoint config: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) GetEndpointConfigByID(ctx context.Context, id int) (*types.EndpointConfig, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Err(err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT id, name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id::text, created_at, updated_at, created_by
		FROM endpoint_configs WHERE id = $1`

	var config types.EndpointConfig
	err = tx.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.Name, &config.MutualAuth, &config.NoEncryption,
		&config.ASLKeyExchangeMethod, &config.Cipher, &config.VersionSetID,
		&config.CreatedAt, &config.UpdatedAt, &config.CreatedBy,
	)
	if err != nil {
		log.Err(err)
		return nil, fmt.Errorf("failed to fetch endpoint config: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Err(err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &config, nil
}

func (s *StateManager) ListEndpointConfigs(ctx context.Context) ([]*types.EndpointConfig, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Err(err).Msgf("Failed to begin transaction")
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `SELECT id, name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id::text, created_at, updated_at, created_by FROM endpoint_configs`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		log.Err(err).Msgf("Failed to execute query")
		return nil, err
	}
	defer rows.Close()

	var configs []*types.EndpointConfig
	for rows.Next() {
		config := new(types.EndpointConfig)
		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.MutualAuth,
			&config.NoEncryption,
			&config.ASLKeyExchangeMethod,
			&config.Cipher,
			&config.VersionSetID,
			&config.CreatedAt,
			&config.UpdatedAt,
			&config.CreatedBy,
		)
		if err != nil {
			log.Err(err).Msgf("Failed to scan row")
			return nil, err
		}
		configs = append(configs, config)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Err(err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return configs, nil
}
