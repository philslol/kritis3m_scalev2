package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

func (s *StateManager) CreateGroup(ctx context.Context, group *types.Group) error {
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
        INSERT INTO groups (
            name, 
            log_level, 
            endpoint_config_name, 
            legacy_config_name,
            version_set_id,
            created_by
        ) VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at, updated_at`

		return tx.QueryRow(ctx, query,
			group.Name,
			group.LogLevel,
			group.EndpointConfigName,
			group.LegacyConfigName,
			group.VersionSetID,
			group.CreatedBy,
		).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create group")
		return err
	}
	return nil
}

func (s *StateManager) GetByID(ctx context.Context, id int) (*types.Group, error) {
	group := &types.Group{}
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
        SELECT 
            id, 
            name, 
            log_level, 
            endpoint_config_name, 
            legacy_config_name,
            version_set_id,
            created_at,
            updated_at,
            created_by
        FROM groups 
        WHERE id = $1`

		err := tx.QueryRow(ctx, query, id).Scan(
			&group.ID,
			&group.Name,
			&group.LogLevel,
			&group.EndpointConfigName,
			&group.LegacyConfigName,
			&group.VersionSetID,
			&group.CreatedAt,
			&group.UpdatedAt,
			&group.CreatedBy,
		)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get group by id")
		return nil, err
	}
	return group, nil
}

// Additional useful methods

func (s *StateManager) GetListGroup(ctx context.Context, versionSetID *uuid.UUID) ([]*types.Group, error) {
	groups := []*types.Group{}
	var err error

	err = s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {

		var query string
		var rows pgx.Rows
		if versionSetID != nil {
			query = `
        SELECT 
            id, 
            name, 
            log_level, 
            endpoint_config_name, 
            legacy_config_name,
            version_set_id,
            created_at,
            updated_at,
            created_by
        FROM groups 
        WHERE version_set_id = $1`
			rows, err = tx.Query(ctx, query, versionSetID)
		} else {
			query = `
        SELECT 
            id, 
            name, 
            log_level, 
            endpoint_config_name, 
            legacy_config_name,
            version_set_id,
            created_at,
            updated_at,
            created_by
        FROM groups 
        `
			rows, err = tx.Query(ctx, query)
		}

		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			group := &types.Group{}
			err := rows.Scan(
				&group.ID,
				&group.Name,
				&group.LogLevel,
				&group.EndpointConfigName,
				&group.LegacyConfigName,
				&group.VersionSetID,
				&group.CreatedAt,
				&group.UpdatedAt,
				&group.CreatedBy,
			)
			if err != nil {
				return err
			}
			groups = append(groups, group)
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get list group")
		return nil, err
	}
	return groups, nil
}

func (s *StateManager) GetGroupByName(ctx context.Context, name string, versionSetID *uuid.UUID) (*types.Group, error) {
	group := &types.Group{}

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT 
			id, 
			name, 
			log_level, 
			endpoint_config_name, 
			legacy_config_name,
			version_set_id,
			created_at,
			updated_at,
			created_by
		FROM groups 
		WHERE name = $1 AND version_set_id = $2`
		err := tx.QueryRow(ctx, query, name, versionSetID).Scan(
			&group.ID,
			&group.Name,
			&group.LogLevel,
			&group.EndpointConfigName,
			&group.LegacyConfigName,
			&group.VersionSetID,
			&group.CreatedAt,
			&group.UpdatedAt,
			&group.CreatedBy,
		)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get group by name")
		return nil, err
	}
	return group, nil
}
