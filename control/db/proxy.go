package db

import (
	"context"
	"log"

	"github.com/philslol/kritis3m_scalev2/control/types"
)

// CreateProxy inserts a new proxy record into the database.
func (s *StateManager) CreateProxy(ctx context.Context, proxy *types.Proxy) (*types.Proxy, error) {
	query := `
	INSERT INTO proxies 
		(node_id, group_id, state, proxy_type, server_endpoint_addr, client_endpoint_addr, version_set_id, state, created_by) 
	VALUES 
		($1, $2, $3, $4, $5, $6, $7, $8, $9) 
	RETURNING id, created_at, updated_at`

	err := s.pool.QueryRow(ctx, query,
		proxy.NodeID, proxy.GroupID, proxy.State, proxy.ProxyType, proxy.ServerEndpointAddr,
		proxy.ClientEndpointAddr, proxy.VersionSetID, proxy.VersionState, proxy.CreatedBy,
	).Scan(&proxy.ID, &proxy.CreatedAt, &proxy.UpdatedAt)

	if err != nil {
		log.Println("Error inserting proxy:", err)
		return nil, err
	}
	return proxy, nil
}

// GetProxyByID retrieves a proxy by ID.
func (s *StateManager) GetProxyByID(ctx context.Context, id int) (*types.Proxy, error) {
	query := `SELECT id, node_id, group_id, state, proxy_type, server_endpoint_addr, 
	client_endpoint_addr, version_set_id, state, created_at, updated_at, created_by FROM proxies WHERE id = $1`

	proxy := &types.Proxy{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&proxy.ID, &proxy.NodeID, &proxy.GroupID, &proxy.State, &proxy.ProxyType,
		&proxy.ServerEndpointAddr, &proxy.ClientEndpointAddr, &proxy.VersionSetID,
		&proxy.VersionState, &proxy.CreatedAt, &proxy.UpdatedAt, &proxy.CreatedBy,
	)

	if err != nil {
		log.Println("Error retrieving proxy:", err)
		return nil, err
	}
	return proxy, nil
}

// UpdateProxy updates an existing proxy record.
func (s *StateManager) UpdateProxy(ctx context.Context, proxy *types.Proxy) error {
	query := `
	UPDATE proxies 
	SET node_id = $1, group_id = $2, state = $3, proxy_type = $4, server_endpoint_addr = $5, 
		client_endpoint_addr = $6, version_set_id = $7, state = $8, updated_at = NOW()
	WHERE id = $9`

	_, err := s.pool.Exec(ctx, query, proxy.NodeID, proxy.GroupID, proxy.State, proxy.ProxyType,
		proxy.ServerEndpointAddr, proxy.ClientEndpointAddr, proxy.VersionSetID, proxy.VersionState, proxy.ID)

	if err != nil {
		log.Println("Error updating proxy:", err)
		return err
	}
	return nil
}

// DeleteProxy removes a proxy by ID.
func (s *StateManager) DeleteProxy(ctx context.Context, id int) error {
	query := `DELETE FROM proxies WHERE id = $1`

	_, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		log.Println("Error deleting proxy:", err)
		return err
	}
	return nil
}

// ListProxies retrieves all proxy records.
func (r *StateManager) ListProxies(ctx context.Context) ([]types.Proxy, error) {
	query := `SELECT id, node_id, group_id, state, proxy_type, server_endpoint_addr, 
	client_endpoint_addr, version_set_id, state, created_at, updated_at, created_by FROM proxies`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		log.Println("Error retrieving proxies:", err)
		return nil, err
	}
	defer rows.Close()

	var proxies []types.Proxy
	for rows.Next() {
		var proxy types.Proxy
		err := rows.Scan(
			&proxy.ID, &proxy.NodeID, &proxy.GroupID, &proxy.State, &proxy.ProxyType,
			&proxy.ServerEndpointAddr, &proxy.ClientEndpointAddr, &proxy.VersionSetID,
			&proxy.VersionState, &proxy.CreatedAt, &proxy.UpdatedAt, &proxy.CreatedBy,
		)
		if err != nil {
			log.Println("Error scanning proxy row:", err)
			return nil, err
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

// GetProxiesByNodeID retrieves proxies associated with a specific node_id.
func (s *StateManager) GetProxiesByNodeID(ctx context.Context, nodeID int) ([]*types.Proxy, error) {
	query := `SELECT id, node_id, group_id, state, proxy_type, server_endpoint_addr, 
	client_endpoint_addr, version_set_id, state, created_at, updated_at, created_by FROM proxies WHERE node_id = $1`

	rows, err := s.pool.Query(ctx, query, nodeID)
	if err != nil {
		log.Println("Error retrieving proxies by node_id:", err)
		return nil, err
	}
	defer rows.Close()

	var proxies []*types.Proxy
	for rows.Next() {
		proxy := new(types.Proxy)
		err := rows.Scan(
			&proxy.ID, &proxy.NodeID, &proxy.GroupID, &proxy.State, &proxy.ProxyType,
			&proxy.ServerEndpointAddr, &proxy.ClientEndpointAddr, &proxy.VersionSetID,
			&proxy.VersionState, &proxy.CreatedAt, &proxy.UpdatedAt, &proxy.CreatedBy,
		)
		if err != nil {
			log.Println("Error scanning proxy row:", err)
			return nil, err
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}
