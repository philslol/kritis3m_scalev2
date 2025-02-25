package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

func (s *StateManager) CreateEndpointConfig(ctx context.Context, config *types.EndpointConfig) error {

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		INSERT INTO endpoint_configs (name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

		return tx.QueryRow(ctx, query,
			config.Name,
			config.MutualAuth,
			config.NoEncryption,
			config.ASLKeyExchangeMethod,
			config.Cipher,
			config.VersionSetID,
			config.CreatedBy,
		).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
	})
	if err != nil {
		log.Err(err).Msg("failed to create endpoint config")
		return err
	}
	return nil
}

func (s *StateManager) GetEndpointConfigByID(ctx context.Context, id int) (*types.EndpointConfig, error) {

	var config types.EndpointConfig
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {

		query := `
		SELECT id, name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id::text, created_at, updated_at, created_by
		FROM endpoint_configs WHERE id = $1`

		return tx.QueryRow(ctx, query, id).Scan(
			&config.ID, &config.Name, &config.MutualAuth, &config.NoEncryption,
			&config.ASLKeyExchangeMethod, &config.Cipher, &config.VersionSetID,
			&config.CreatedAt, &config.UpdatedAt, &config.CreatedBy,
		)
	})
	if err != nil {
		log.Err(err).Msg("failed to get endpoint config by id")
		return nil, err
	}
	return &config, nil
}

func (s *StateManager) ListEndpointConfigs(ctx context.Context, versionSetID *uuid.UUID) ([]*types.EndpointConfig, error) {
	var configs []*types.EndpointConfig

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {

		var query string
		if versionSetID != nil {
			query = `SELECT id, name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id::text, created_at, updated_at, created_by FROM endpoint_configs WHERE version_set_id = $1`
		} else {
			query = `SELECT id, name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id::text, created_at, updated_at, created_by FROM endpoint_configs`
		}

		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

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
				return err
			}
			configs = append(configs, config)
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to list endpoint configs")
		return nil, err
	}

	return configs, nil
}

func (s *StateManager) GetEndpointConfigByName(ctx context.Context, name string, versionSetID *uuid.UUID) (*types.EndpointConfig, error) {

	var config types.EndpointConfig
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {

		query := `
		SELECT id, name, mutual_auth, no_encryption, asl_key_exchange_method, cipher, version_set_id::text, created_at, updated_at, created_by
		FROM endpoint_configs WHERE name = $1 AND version_set_id = $2`

		return tx.QueryRow(ctx, query, name, versionSetID).Scan(
			&config.ID, &config.Name, &config.MutualAuth, &config.NoEncryption,
			&config.ASLKeyExchangeMethod, &config.Cipher, &config.VersionSetID,
			&config.CreatedAt, &config.UpdatedAt, &config.CreatedBy,
		)
	})
	if err != nil {
		log.Err(err).Msg("failed to get endpoint config by name")
		return nil, err
	}

	return &config, nil
}
