package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
)

func (s *StateManager) NodeUpdate(SerialNumber string, VersionSet string, ctx context.Context) (*v1.NodeUpdateItem, error) {
	node := &v1.NodeUpdateItem{
		SerialNumber: SerialNumber,
		VersionSetId: VersionSet,
	}
	groupMap := make(map[string]*v1.GroupProxyUpdate)
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
		rows, err := tx.Query(ctx, query, SerialNumber, VersionSet)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				groupName      string
				groupLogLevel  int32
				proxyID        int32
				proxyName      string
				proxyState     bool
				proxyType      int32
				serverEndpoint string
				clientEndpoint string
				endpointConfig types.EndpointConfig
				legacyConfig   types.EndpointConfig
			)

			err := rows.Scan(
				&node.SerialNumber, &node.NetworkIndex, &node.Locality, &node.VersionSetId,
				&groupName, &groupLogLevel,
				&endpointConfig.Name, &endpointConfig.MutualAuth, &endpointConfig.NoEncryption, &endpointConfig.ASLKeyExchangeMethod, &endpointConfig.Cipher,
				&legacyConfig.Name, &legacyConfig.MutualAuth, &legacyConfig.NoEncryption, &legacyConfig.ASLKeyExchangeMethod, &legacyConfig.Cipher,
				&proxyID, &proxyName, &proxyState, &proxyType, &serverEndpoint, &clientEndpoint,
			)
			if err != nil {
				return err
			}

			groupKey := groupName
			if groupKey == "" {
				continue
			}

			if _, exists := groupMap[groupKey]; !exists {
				groupMap[groupKey] = &v1.GroupProxyUpdate{
					GroupName:     groupName,
					GroupLogLevel: groupLogLevel,
					EndpointConfig: &v1.EndpointConfig{
						Name:                 endpointConfig.Name,
						MutualAuth:           endpointConfig.MutualAuth,
						NoEncryption:         endpointConfig.NoEncryption,
						AslKeyExchangeMethod: v1.AslKeyexchangeMethod(v1.AslKeyexchangeMethod_value[endpointConfig.ASLKeyExchangeMethod]),
						Cipher:               endpointConfig.Cipher,
					},
					LegacyConfig: &v1.EndpointConfig{
						Name:                 legacyConfig.Name,
						MutualAuth:           legacyConfig.MutualAuth,
						NoEncryption:         legacyConfig.NoEncryption,
						AslKeyExchangeMethod: v1.AslKeyexchangeMethod(v1.AslKeyexchangeMethod_value[legacyConfig.ASLKeyExchangeMethod]),
						Cipher:               legacyConfig.Cipher,
					},
					Proxies: []*v1.UpdateProxy{},
				}
			}

			groupMap[groupKey].Proxies = append(groupMap[groupKey].Proxies, &v1.UpdateProxy{
				Name:               proxyName,
				ServerEndpointAddr: serverEndpoint,
				ClientEndpointAddr: clientEndpoint,
				ProxyType:          v1.ProxyType(proxyType),
			})
		}

		for _, group := range groupMap {
			node.GroupProxyUpdate = append(node.GroupProxyUpdate, group)
		}

		return nil
	})

	if len(node.GroupProxyUpdate) == 0 {
		return nil, nil
	}
	return node, nil
}
