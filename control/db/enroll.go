package db

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (s *StateManager) ListEnroll(ctx context.Context) ([]*types.EnrollCallRequest, error) {
	var enrolls []*types.EnrollCallRequest

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT id, est_serial_number, serial_number, organization, issued_at, expires_at, 
		       signature_algorithm, plane, created_at, updated_at
		FROM enroll
		ORDER BY created_at DESC`

		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			enroll := new(types.EnrollCallRequest)
			err := rows.Scan(
				&enroll.ID,
				&enroll.EstSerialNumber,
				&enroll.SerialNumber,
				&enroll.Organization,
				&enroll.IssuedAt,
				&enroll.ExpiresAt,
				&enroll.SignatureAlgorithm,
				&enroll.Plane,
				&enroll.CreatedAt,
				&enroll.UpdatedAt,
			)
			if err != nil {
				return err
			}
			enrolls = append(enrolls, enroll)
		}
		return nil
	})

	if err != nil {
		log.Err(err).Msg("failed to list enroll requests")
		return nil, err
	}

	return enrolls, nil
}

func (s *StateManager) GetEnroll(ctx context.Context, serialNumber string) ([]*types.EnrollCallRequest, error) {
	var enrolls []*types.EnrollCallRequest

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT id, est_serial_number, serial_number, organization, issued_at, expires_at, 
		       signature_algorithm, plane, created_at, updated_at
		FROM enroll
		WHERE serial_number = $1
		ORDER BY created_at DESC`

		rows, err := tx.Query(ctx, query, serialNumber)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			enroll := new(types.EnrollCallRequest)
			err := rows.Scan(
				&enroll.ID,
				&enroll.EstSerialNumber,
				&enroll.SerialNumber,
				&enroll.Organization,
				&enroll.IssuedAt,
				&enroll.ExpiresAt,
				&enroll.SignatureAlgorithm,
				&enroll.Plane,
				&enroll.CreatedAt,
				&enroll.UpdatedAt,
			)
			if err != nil {
				return err
			}
			enrolls = append(enrolls, enroll)
		}
		return nil
	})

	if err != nil {
		log.Err(err).Msg("failed to get enroll requests by serial number")
		return nil, err
	}

	return enrolls, nil
}

func (s *StateManager) GetEnrollEstSerial(ctx context.Context, estSerialNumber string) (*types.EnrollCallRequest, error) {
	var enroll types.EnrollCallRequest

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT id, est_serial_number, serial_number, organization, issued_at, expires_at, 
		       signature_algorithm, plane, created_at, updated_at
		FROM enroll
		WHERE est_serial_number = $1`

		return tx.QueryRow(ctx, query, estSerialNumber).Scan(
			&enroll.ID,
			&enroll.EstSerialNumber,
			&enroll.SerialNumber,
			&enroll.Organization,
			&enroll.IssuedAt,
			&enroll.ExpiresAt,
			&enroll.SignatureAlgorithm,
			&enroll.Plane,
			&enroll.CreatedAt,
			&enroll.UpdatedAt,
		)
	})

	if err != nil {
		log.Err(err).Msg("failed to get enroll request by EST serial number")
		return nil, err
	}

	return &enroll, nil
}

func (s *StateManager) CreateEnroll(ctx context.Context, enroll *types.EnrollCallRequest) error {
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		INSERT INTO enroll 
		(est_serial_number, serial_number, organization, issued_at, expires_at, 
		 signature_algorithm, plane)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

		return tx.QueryRow(ctx, query,
			enroll.EstSerialNumber,
			enroll.SerialNumber,
			enroll.Organization,
			enroll.IssuedAt,
			enroll.ExpiresAt,
			enroll.SignatureAlgorithm,
			enroll.Plane,
		).Scan(&enroll.ID, &enroll.CreatedAt, &enroll.UpdatedAt)
	})

	if err != nil {
		log.Err(err).Msg("failed to create enroll request")
		return err
	}

	return nil
}

// must be modified with optional arguments
func (s *StateManager) UpdateEnroll(ctx context.Context, enroll *types.EnrollCallRequest) error {
	updates := map[string]any{
		"est_serial_number":   enroll.EstSerialNumber,
		"organization":        enroll.Organization,
		"issued_at":           enroll.IssuedAt,
		"expires_at":          enroll.ExpiresAt,
		"signature_algorithm": enroll.SignatureAlgorithm,
		"plane":               enroll.Plane,
	}

	return s.Update(ctx, "enroll", updates, "id", enroll.ID)
}

func (s *StateManager) DeleteEnroll(ctx context.Context, id int) error {
	return s.Delete(ctx, "enroll", "id", strconv.Itoa(id))
}
