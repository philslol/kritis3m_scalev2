package db

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

// CreateProxy inserts a new proxy record into the database.
func (s *StateManager) CreateProxy(ctx context.Context, proxy *types.Proxy) (*types.Proxy, error) {

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		INSERT INTO proxies 
			(name, node_name, group_name, state, proxy_type, server_endpoint_addr, client_endpoint_addr, version_set_id, created_by) 
	VALUES 
		($1, $2, $3, $4, $5, $6, $7, $8) 
	RETURNING id, created_at, updated_at`

		return s.pool.QueryRow(ctx, query,
			proxy.Name, proxy.NodeSerial, proxy.GroupName, proxy.State, proxy.ProxyType, proxy.ServerEndpointAddr,
			proxy.ClientEndpointAddr, proxy.VersionSetID, proxy.CreatedBy,
		).Scan(&proxy.ID, &proxy.CreatedAt, &proxy.UpdatedAt)
	})
	if err != nil {
		log.Err(err).Msg("failed to create proxy")
		return nil, err
	}

	return proxy, nil
}

// ListProxies retrieves all proxy records.
func (s *StateManager) ListProxies(ctx context.Context) ([]types.Proxy, error) {

	var proxies []types.Proxy

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `SELECT id, name, node_name, group_name, state, proxy_type, server_endpoint_addr, 
	client_endpoint_addr, version_set_id,  created_at, updated_at, created_by FROM proxies`

		rows, err := s.pool.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var proxy types.Proxy
			err := rows.Scan(
				&proxy.ID, &proxy.Name, &proxy.NodeSerial, &proxy.GroupName, &proxy.State, &proxy.ProxyType,
				&proxy.ServerEndpointAddr, &proxy.ClientEndpointAddr, &proxy.VersionSetID,
				&proxy.CreatedAt, &proxy.UpdatedAt, &proxy.CreatedBy,
			)
			if err != nil {
				return err
			}
			proxies = append(proxies, proxy)
		}
		return nil
	})

	if err != nil {
		log.Err(err).Msg("failed to list proxies")
		return nil, err
	}

	return proxies, nil
}

// GetProxiesByNodeID retrieves proxies associated with a specific node_id.
func (s *StateManager) GetProxiesByProxyID(ctx context.Context, proxyID int) ([]*types.Proxy, error) {

	var proxies []*types.Proxy

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {

		query := `SELECT id, name, node_name, group_name, state, proxy_type, server_endpoint_addr, 
	client_endpoint_addr, version_set_id, created_at, updated_at, created_by FROM proxies WHERE id = $1`

		rows, err := s.pool.Query(ctx, query, proxyID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			proxy := new(types.Proxy)
			err := rows.Scan(
				&proxy.ID, &proxy.Name, &proxy.NodeSerial, &proxy.GroupName, &proxy.State, &proxy.ProxyType,
				&proxy.ServerEndpointAddr, &proxy.ClientEndpointAddr, &proxy.VersionSetID,
				&proxy.CreatedAt, &proxy.UpdatedAt, &proxy.CreatedBy,
			)
			if err != nil {
				return err
			}
			proxies = append(proxies, proxy)
		}
		return nil
	})
	if err != nil {
		log.Err(err).Msg("failed to get proxies by proxy id")
		return nil, err
	}

	return proxies, nil
}

// get by proxy name and version set id
func (s *StateManager) GetProxyByName(ctx context.Context, name string, versionSetID uuid.UUID) (*types.Proxy, error) {

	proxy := &types.Proxy{}
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT id, node_serial, group_name, state, proxy_type, server_endpoint_addr, client_endpoint_addr, version_set_id, created_by
		FROM proxies
		WHERE name = $1 AND version_set_id = $2`

		row := s.pool.QueryRow(ctx, query, name, versionSetID)

		return row.Scan(
			&proxy.ID,
			&proxy.NodeSerial,
			&proxy.GroupName,
			&proxy.State,
			&proxy.ProxyType,
			&proxy.ServerEndpointAddr,
			&proxy.ClientEndpointAddr,
			&proxy.VersionSetID,
			&proxy.CreatedBy,
		)
	})

	if err != nil {
		log.Err(err).Msg("failed to get proxy by name and version set id")
		return nil, err
	}

	return proxy, nil
}

func (s *StateManager) GetProxyBySerialNumber(ctx context.Context, serialNumber string, versionSetID uuid.UUID) ([]*types.Proxy, error) {

	var proxies []*types.Proxy
	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT id, node_serial, group_name, state, proxy_type, server_endpoint_addr, client_endpoint_addr, version_set_id, created_by
		FROM proxies	
		WHERE serial_number = $1 AND version_set_id = $2`

		rows, err := s.pool.Query(ctx, query, serialNumber, versionSetID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			proxy := &types.Proxy{}
			err := rows.Scan(
				&proxy.ID,
				&proxy.NodeSerial,
				&proxy.GroupName,
				&proxy.State,
				&proxy.ProxyType,
				&proxy.ServerEndpointAddr,
				&proxy.ClientEndpointAddr,
				&proxy.VersionSetID,
				&proxy.CreatedBy,
			)
			if err != nil {
				return err

			}
			proxies = append(proxies, proxy)
		}
		return nil
	})

	if err != nil {
		log.Err(err).Msg("failed to get proxies by serial number")
		return nil, err
	}

	return proxies, nil

}

// get by proxy id
func (s *StateManager) GetProxyByID(ctx context.Context, id int) (*types.Proxy, error) {

	proxy := &types.Proxy{}

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		query := `
		SELECT id, node_serial, group_name, state, proxy_type, server_endpoint_addr, client_endpoint_addr, version_set_id, created_by
		FROM proxies
		WHERE id = $1`

		return s.pool.QueryRow(ctx, query, id).Scan(
			&proxy.ID,
			&proxy.NodeSerial,
			&proxy.GroupName,
			&proxy.State,
			&proxy.ProxyType,
			&proxy.ServerEndpointAddr,
			&proxy.ClientEndpointAddr,
			&proxy.VersionSetID,
			&proxy.CreatedBy,
		)

	})

	if err != nil {
		log.Err(err).Msg("failed to get proxy by id")
		return nil, err
	}

	return proxy, nil

}
