package db

import (
	"context"
	"fmt"

	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (s *StateManager) CreateProxy(ctx context.Context, proxy *types.Proxy) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO proxies (
			 node_id, group_id, state, proxy_type,
			server_endpoint_addr, client_endpoint_addr, status,
			version, previous_version_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRow(ctx, query,
		proxy.NodeID, proxy.GroupID, proxy.State,
		proxy.ProxyType, proxy.ServerEndpointAddr, proxy.ClientEndpointAddr,
		proxy.Status, proxy.Version, proxy.PreviousVersionID, proxy.CreatedBy,
	).Scan(&proxy.ID, &proxy.CreatedAt, &proxy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert proxy: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) GetProxy(ctx context.Context, id int) (*types.Proxy, error) {
	proxy := &types.Proxy{}
	query := `
		SELECT id,  node_id, group_id, state,
			   proxy_type, server_endpoint_addr, client_endpoint_addr,
			   status, version, previous_version_id, created_at,
			   updated_at, created_by
		FROM proxies WHERE id = $1`

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&proxy.ID, &proxy.NodeID, &proxy.GroupID,
		&proxy.State, &proxy.ProxyType, &proxy.ServerEndpointAddr,
		&proxy.ClientEndpointAddr, &proxy.Status, &proxy.Version,
		&proxy.PreviousVersionID, &proxy.CreatedAt, &proxy.UpdatedAt,
		&proxy.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy: %w", err)
	}
	return proxy, nil
}

func (s *StateManager) UpdateProxy(ctx context.Context, proxy *types.Proxy) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE proxies
		SET state = $1, proxy_type = $2, server_endpoint_addr = $3,
			client_endpoint_addr = $4, status = $5, version = $6,
			previous_version_id = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at`

	err = tx.QueryRow(ctx, query,
		proxy.State, proxy.ProxyType, proxy.ServerEndpointAddr,
		proxy.ClientEndpointAddr, proxy.Status, proxy.Version,
		proxy.PreviousVersionID, proxy.ID,
	).Scan(&proxy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update proxy: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *StateManager) DeleteProxy(ctx context.Context, id int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM proxies WHERE id = $1`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete proxy: %w", err)
	}

	return tx.Commit(ctx)
}
