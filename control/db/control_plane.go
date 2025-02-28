package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

type NodeUpdateItem struct {
	SerialNumber     string
	NetworkIndex     int
	Locality         string
	VersionSetId     string
	GroupProxyUpdate []GroupProxyUpdate
}
type GroupProxyUpdate struct {
	GroupName      string
	GroupLogLevel  int
	EndpointConfig types.EndpointConfig
	LegacyConfig   *types.EndpointConfig
	Proxies        []*types.Proxy
}

func (s *StateManager) NodeUpdate() {

	node := &NodeUpdateItem{}
	groupMap := make(map[string]*GroupProxyUpdate)
	query := `
	WITH node_info AS (
	    SELECT
	        serial_number,
	        network_index,
	        locality,
	        version_set_id::text
	    FROM
	        nodes
	    WHERE
	        serial_number = $1
	        AND version_set_id = $2::uuid
	),
	node_groups AS (
	    SELECT DISTINCT
	        p.group_name
	    FROM
	        proxies p
	    WHERE
	        p.node_serial = $1
	        AND p.version_set_id = $2::uuid
	)
	SELECT
	    -- Node information
	    n.serial_number,
	    n.network_index,
	    n.locality,
	    n.version_set_id,

	    -- Group information
	    g.name AS group_name,
	    g.log_level AS group_log_level,

	    -- Endpoint Config
	    ec1.name AS endpoint_config_name,
	    ec1.mutual_auth AS endpoint_mutual_auth,
	    ec1.no_encryption AS endpoint_no_encryption,
	    ec1.asl_key_exchange_method AS endpoint_kex_method,
	    ec1.cipher AS endpoint_cipher,

	    -- Legacy Config
	    ec2.name AS legacy_config_name,
	    ec2.mutual_auth AS legacy_mutual_auth,
	    ec2.no_encryption AS legacy_no_encryption,
	    ec2.asl_key_exchange_method AS legacy_kex_method,
	    ec2.cipher AS legacy_cipher,

	    -- Proxy information
	    p.id AS proxy_id,
	    p.name AS proxy_name,
	    p.state AS proxy_state,
	    p.proxy_type,
	    p.server_endpoint_addr,
	    p.client_endpoint_addr
	FROM
	    node_info n
	JOIN
	    node_groups ng ON true
	JOIN
	    groups g ON ng.group_name = g.name AND g.version_set_id = $2::uuid
	LEFT JOIN
	    endpoint_configs ec1 ON g.endpoint_config_name = ec1.name AND g.version_set_id = ec1.version_set_id
	LEFT JOIN
	    endpoint_configs ec2 ON g.legacy_config_name = ec2.name AND g.version_set_id = ec2.version_set_id
	LEFT JOIN
	    proxies p ON g.name = p.group_name AND g.version_set_id = p.version_set_id AND p.node_serial = $1;
	`

	s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(query, serialNumber, versionSetID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				groupName      sql.NullString
				groupLogLevel  sql.NullInt32
				proxyID        sql.NullInt32
				proxyName      sql.NullString
				proxyState     sql.NullInt32
				proxyType      sql.NullInt32
				serverEndpoint sql.NullString
				clientEndpoint sql.NullString
				endpointConfig EndpointConfig
				legacyConfig   EndpointConfig
			)

			err := rows.Scan(
				&node.SerialNumber, &node.NetworkIndex, &node.Locality, &node.VersionSetId,
				&groupName, &groupLogLevel,
				&endpointConfig.Name, &endpointConfig.MutualAuth, &endpointConfig.NoEncryption, &endpointConfig.KexMethod, &endpointConfig.Cipher,
				&legacyConfig.Name, &legacyConfig.MutualAuth, &legacyConfig.NoEncryption, &legacyConfig.KexMethod, &legacyConfig.Cipher,
				&proxyID, &proxyName, &proxyState, &proxyType, &serverEndpoint, &clientEndpoint,
			)
			if err != nil {
				return nil, err
			}

			groupKey := groupName.String
			if groupKey == "" {
				continue
			}

			if _, exists := groupMap[groupKey]; !exists {
				groupMap[groupKey] = &GroupProxyUpdate{
					GroupName:      groupName.String,
					GroupLogLevel:  groupLogLevel.Int32,
					EndpointConfig: &endpointConfig,
					LegacyConfig:   legacyConfig,
					Proxies:        []UpdateProxy{},
				}
			}

			if proxyID.Valid {
				groupMap[groupKey].Proxies = append(groupMap[groupKey].Proxies, UpdateProxy{
					ID:             proxyID.Int32,
					Name:           proxyName.String,
					State:          proxyState.Int32,
					ProxyType:      proxyType.Int32,
					ServerEndpoint: serverEndpoint.String,
					ClientEndpoint: clientEndpoint.String,
				})
			}
		}

		for _, group := range groupMap {
			node.GroupProxyUpdate = append(node.GroupProxyUpdate, group)
		}

		return nil
	})

}
