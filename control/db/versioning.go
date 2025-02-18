package db

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

// VersionSet represents a row in the version_sets table.
// CreateVersionSet inserts a new version set into the database.
func (s *StateManager) CreateVersionSet(ctx context.Context, vs types.VersionSet) (uuid.UUID, error) {
	var id uuid.UUID
	query := `
		INSERT INTO version_sets (name, description, state, created_by, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	err := s.pool.QueryRow(ctx, query, vs.Name, vs.Description, vs.State, vs.CreatedBy, vs.Metadata).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// GetVersionSetByID retrieves a version set by its ID.
func (s *StateManager) GetVersionSetByID(ctx context.Context, id uuid.UUID) (*types.VersionSet, error) {
	query := `
		SELECT id, name, description, state, created_at, activated_at, disabled_at, created_by, metadata
		FROM version_sets WHERE id = $1`
	row := s.pool.QueryRow(ctx, query, id)

	var vs types.VersionSet
	err := row.Scan(&vs.ID, &vs.Name, &vs.Description, &vs.State, &vs.CreatedAt,
		&vs.ActivatedAt, &vs.DisabledAt, &vs.CreatedBy, &vs.Metadata)
	if err != nil {
		return nil, err
	}
	return &vs, nil
}

// UpdateVersionSet updates an existing version set.
func (s *StateManager) UpdateVersionSet(ctx context.Context, vs types.VersionSet) error {
	query := `
		UPDATE version_sets 
		SET name = $1, description = $2, state = $3, activated_at = $4, disabled_at = $5, metadata = $6
		WHERE id = $7`
	_, err := s.pool.Exec(ctx, query, vs.Name, vs.Description, vs.State, vs.ActivatedAt, vs.DisabledAt, vs.Metadata, vs.ID)
	return err
}

// DeleteVersionSet removes a version set from the database.
func (s *StateManager) DeleteVersionSet(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM version_sets WHERE id = $1`
	_, err := s.pool.Exec(ctx, query, id)
	return err
}

// ListVersionSets retrieves all version sets.
func (s *StateManager) ListVersionSets(ctx context.Context) ([]*types.VersionSet, error) {
	query := `
		SELECT id, name, description, state, created_at, activated_at, disabled_at, created_by, metadata
		FROM version_sets`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versionSets []*types.VersionSet
	for rows.Next() {
		vs := new(types.VersionSet)
		err := rows.Scan(&vs.ID, &vs.Name, &vs.Description, &vs.State, &vs.CreatedAt,
			&vs.ActivatedAt, &vs.DisabledAt, &vs.CreatedBy, &vs.Metadata)
		if err != nil {
			return nil, err
		}
		versionSets = append(versionSets, vs)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return versionSets, nil
}

/*----------------------------- VERSION TRANSITION -----------------------------------------*/

func (s *StateManager) CreateVersionTransition(ctx context.Context, transition *types.VersionTransition) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO version_transitions (from_version_id, to_version_id, status, created_by, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, started_at`

	err = tx.QueryRow(ctx, query,
		transition.FromVersionID,
		transition.ToVersionID,
		transition.Status,
		transition.CreatedBy,
		transition.Metadata,
	).Scan(&transition.ID, &transition.StartedAt)
	if err != nil {
		return fmt.Errorf("failed to insert version transition: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) GetVersionTransitionByID(ctx context.Context, id string) (*types.VersionTransition, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT id, from_version_id, to_version_id, status, started_at, completed_at, created_by, metadata
		FROM version_transitions WHERE id = $1`

	var vt types.VersionTransition
	err = tx.QueryRow(ctx, query, id).Scan(
		&vt.ID, &vt.FromVersionID, &vt.ToVersionID, &vt.Status,
		&vt.StartedAt, &vt.CompletedAt, &vt.CreatedBy, &vt.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version transition: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &vt, nil
}

func (s *StateManager) ListVersionTransitions(ctx context.Context) ([]*types.VersionTransition, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `SELECT id, from_version_id, to_version_id, status, started_at, completed_at, created_by, metadata FROM version_transitions`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var transitions []*types.VersionTransition
	for rows.Next() {
		vt := new(types.VersionTransition)
		err := rows.Scan(
			vt.ID, vt.FromVersionID, vt.ToVersionID, vt.Status,
			vt.StartedAt, vt.CompletedAt, vt.CreatedBy, vt.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		transitions = append(transitions, vt)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return transitions, nil
}

func (s *StateManager) DeleteVersionTransition(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM version_transitions WHERE id = $1`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete version transition: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) UpdateVersionTransition(ctx context.Context, transition *types.VersionTransition) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE version_transitions 
		SET status = $1, completed_at = $2, metadata = $3 
		WHERE id = $4`

	_, err = tx.Exec(ctx, query, transition.Status, transition.CompletedAt, transition.Metadata, transition.ID)
	if err != nil {
		return fmt.Errorf("failed to update version transition: %w", err)
	}

	return tx.Commit(ctx)
}
