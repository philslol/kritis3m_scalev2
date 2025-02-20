package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (s *StateManager) Create(ctx context.Context, group *types.Group) error {
	query := `
        INSERT INTO groups (
            name, 
            log_level, 
            endpoint_config_id, 
            legacy_config_id,
            state,
            version_set_id,
            created_by
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at, updated_at`

	return s.pool.QueryRow(ctx, query,
		group.Name,
		group.LogLevel,
		group.EndpointConfigID,
		group.LegacyConfigID,
		group.State,
		group.VersionSetID,
		group.CreatedBy,
	).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
}

func (s *StateManager) GetByID(ctx context.Context, id int) (*types.Group, error) {
	group := &types.Group{}
	query := `
        SELECT 
            id, 
            name, 
            log_level, 
            endpoint_config_id, 
            legacy_config_id,
            state,
            version_set_id,
            created_at,
            updated_at,
            created_by
        FROM groups 
        WHERE id = $1`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&group.ID,
		&group.Name,
		&group.LogLevel,
		&group.EndpointConfigID,
		&group.LegacyConfigID,
		&group.State,
		&group.VersionSetID,
		&group.CreatedAt,
		&group.UpdatedAt,
		&group.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// Additional useful methods

func (s *StateManager) GetByVersionSetID(ctx context.Context, versionSetID uuid.UUID) ([]*types.Group, error) {
	groups := []*types.Group{}
	query := `
        SELECT 
            id, 
            name, 
            log_level, 
            endpoint_config_id, 
            legacy_config_id,
            state,
            version_set_id,
            created_at,
            updated_at,
            created_by
        FROM groups 
        WHERE version_set_id = $1`

	rows, err := s.pool.Query(ctx, query, versionSetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		group := &types.Group{}
		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.LogLevel,
			&group.EndpointConfigID,
			&group.LegacyConfigID,
			&group.State,
			&group.VersionSetID,
			&group.CreatedAt,
			&group.UpdatedAt,
			&group.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *StateManager) GetByState(ctx context.Context, state types.VersionState) ([]*types.Group, error) {
	groups := []*types.Group{}
	query := `
        SELECT 
            id, 
            name, 
            log_level, 
            endpoint_config_id, 
            legacy_config_id,
            state,
            version_set_id,
            created_at,
            updated_at,
            created_by
        FROM groups 
        WHERE state = $1`

	rows, err := s.pool.Query(ctx, query, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		group := new(types.Group)
		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.LogLevel,
			&group.EndpointConfigID,
			&group.LegacyConfigID,
			&group.State,
			&group.VersionSetID,
			&group.CreatedAt,
			&group.UpdatedAt,
			&group.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}
