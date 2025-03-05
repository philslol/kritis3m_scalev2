package db

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

// VersionSet represents a row in the version_sets table.
// CreateVersionSet inserts a new version set into the database.
func (s *StateManager) CreateVersionSet(ctx context.Context, vs types.VersionSet) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO version_sets (name, description, created_by, metadata)
			VALUES ($1, $2, $3, $4)
			RETURNING id`
		return tx.QueryRow(ctx, query, vs.Name, vs.Description, vs.CreatedBy, vs.Metadata).Scan(&id)
	})
	if err != nil {
		log.Err(err).Msg("failed to create version set")
		return uuid.Nil, fmt.Errorf("failed to create version set: %w", err)
	}
	return id, nil
}

// DeleteVersionSet soft deletes a version set by setting its disabled_at timestamp.
func (s *StateManager) DeleteVersionSet(ctx context.Context, id uuid.UUID) error {
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
			UPDATE version_sets 
			SET disabled_at = NOW()
			WHERE id = $1 AND disabled_at IS NULL`
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return fmt.Errorf("failed to delete version set: %w", err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("version set not found or already disabled")
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to delete version set")
		return err
	}
	return nil
}

// GetVersionSetByID retrieves a version set by its ID.
func (s *StateManager) GetVersionSetByID(ctx context.Context, id uuid.UUID) (*types.VersionSet, error) {
	var vs types.VersionSet
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id::text, name, description, state, created_at, activated_at, disabled_at, created_by, metadata
			FROM version_sets WHERE id = $1::uuid`
		var idStr string
		err := tx.QueryRow(ctx, query, id.String()).Scan(
			&idStr, &vs.Name, &vs.Description, &vs.State, &vs.CreatedAt,
			&vs.ActivatedAt, &vs.DisabledAt, &vs.CreatedBy, &vs.Metadata)
		if err != nil {
			return err
		}
		vs.ID, err = uuid.FromString(idStr)
		return err
	})
	if err != nil {
		log.Err(err).Msg("failed to get version set by id")
		return nil, err
	}
	return &vs, nil
}

// ListVersionSets retrieves all active version sets.
func (s *StateManager) ListVersionSets(ctx context.Context) ([]*types.VersionSet, error) {
	var versionSets []*types.VersionSet
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, name, description, state, created_at, activated_at, disabled_at, created_by, metadata
			FROM version_sets
			WHERE disabled_at IS NULL
			ORDER BY created_at DESC`
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			vs := new(types.VersionSet)
			if err := rows.Scan(
				&vs.ID, &vs.Name, &vs.Description, &vs.State, &vs.CreatedAt,
				&vs.ActivatedAt, &vs.DisabledAt, &vs.CreatedBy, &vs.Metadata); err != nil {
				return err
			}
			versionSets = append(versionSets, vs)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list version sets: %w", err)
	}
	return versionSets, nil
}

// ActivateVersionSet marks a version set as active.
func (s *StateManager) ActivateVersionSet(ctx context.Context, id uuid.UUID) error {
	return s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
			UPDATE version_sets 
			SET activated_at = NOW(), state = 'active'
			WHERE id = $1 AND disabled_at IS NULL AND activated_at IS NULL`
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return fmt.Errorf("failed to activate version set: %w", err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("version set not found, already activated, or disabled")
		}
		return nil
	})
}

/*----------------------------- VERSION TRANSITION -----------------------------------------*/

func (s *StateManager) CreateVersionTransition(ctx context.Context, transition *types.VersionTransition) error {
	return s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
			INSERT INTO version_transitions (from_version_transition, to_version_id, status, created_by, metadata)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, started_at`

		return tx.QueryRow(ctx, query,
			transition.FromVersionTransition,
			transition.ToVersionSetID,
			transition.Status,
			transition.CreatedBy,
			transition.Metadata,
		).Scan(&transition.ID, &transition.StartedAt)
	})
}

func (s *StateManager) GetVersionTransitionByID(ctx context.Context, id string) (*types.VersionTransition, error) {
	var vt types.VersionTransition
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
			SELECT id, from_version_transition, to_version_id, status, started_at, completed_at, created_by, metadata
			FROM version_transitions WHERE id = $1`

		return tx.QueryRow(ctx, query, id).Scan(
			&vt.ID, &vt.FromVersionTransition, &vt.ToVersionSetID, &vt.Status,
			&vt.StartedAt, &vt.CompletedAt, &vt.CreatedBy, &vt.Metadata,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version transition: %w", err)
	}
	return &vt, nil
}

// func (s *StateManager) ListVersionTransitions(ctx context.Context) ([]*types.VersionTransition, error) {
// 	var transitions []*types.VersionTransition
// 	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
// 		query := `SELECT id, from_version_id, to_version_id, status, started_at, completed_at, created_by, metadata
// 			FROM version_transitions`
// 		rows, err := tx.Query(ctx, query)
// 		if err != nil {
// 			return fmt.Errorf("failed to execute query: %w", err)
// 		}
// 		defer rows.Close()

// 		for rows.Next() {
// 			vt := new(types.VersionTransition)
// 			err := rows.Scan(
// 				&vt.ID, &vt.FromVersionID, &vt.ToVersionID, &vt.Status,
// 				&vt.StartedAt, &vt.CompletedAt, &vt.CreatedBy, &vt.Metadata,
// 			)
// 			if err != nil {
// 				return fmt.Errorf("failed to scan row: %w", err)
// 			}
// 			transitions = append(transitions, vt)
// 		}
// 		return rows.Err()
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to list version transitions: %w", err)
// 	}
// 	return transitions, nil
// }

// // UpdateVersionSet updates an existing version set
// func (s *StateManager) UpdateVersionSet(ctx context.Context, vs types.VersionSet) error {
// 	return s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
// 		query := `
// 			UPDATE version_sets
// 			SET name = $1, description = $2, state = $3, metadata = $4
// 			WHERE id = $5 AND disabled_at IS NULL`
// 		result, err := tx.Exec(ctx, query, vs.Name, vs.Description, vs.State, vs.Metadata, vs.ID)
// 		if err != nil {
// 			return fmt.Errorf("failed to update version set: %w", err)
// 		}

// 		if result.RowsAffected() == 0 {
// 			return fmt.Errorf("version set not found or already disabled")
// 		}
// 		return nil
// 	})
// }
