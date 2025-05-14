package db

import (
	"context"
	"fmt"

	grpc_control_plane "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/control_plane"
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"

	// v1 "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/gen/go/v1"
	"github.com/jackc/pgx/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

func (s *StateManager) NodeUpdate(SerialNumber string, VersionSet string, ctx context.Context) (*grpc_control_plane.NodeUpdateItem, error) {
	node := &grpc_control_plane.NodeUpdateItem{
		SerialNumber: SerialNumber,
		VersionSetId: VersionSet,
	}
	groupMap := make(map[string]*grpc_control_plane.GroupProxyUpdate)
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
				proxyType      string
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
				log.Error().Err(err).Msg("Failed to scan row")
				return err
			}

			groupKey := groupName
			if groupKey == "" {
				continue
			}

			if _, exists := groupMap[groupKey]; !exists {
				groupMap[groupKey] = &grpc_control_plane.GroupProxyUpdate{
					GroupName:     groupName,
					GroupLogLevel: groupLogLevel,
					EndpointConfig: &grpc_southbound.EndpointConfig{
						Name:                 endpointConfig.Name,
						MutualAuth:           endpointConfig.MutualAuth,
						NoEncryption:         endpointConfig.NoEncryption,
						AslKeyExchangeMethod: endpointConfig.ASLKeyExchangeMethod,
						Cipher:               endpointConfig.Cipher,
					},
					LegacyConfig: &grpc_southbound.EndpointConfig{
						Name:                 legacyConfig.Name,
						MutualAuth:           legacyConfig.MutualAuth,
						NoEncryption:         legacyConfig.NoEncryption,
						AslKeyExchangeMethod: legacyConfig.ASLKeyExchangeMethod,
						Cipher:               legacyConfig.Cipher,
					},
					Proxies: []*grpc_control_plane.UpdateProxy{},
				}
			}

			groupMap[groupKey].Proxies = append(groupMap[groupKey].Proxies, &grpc_control_plane.UpdateProxy{
				Name:               proxyName,
				ServerEndpointAddr: serverEndpoint,
				ClientEndpointAddr: clientEndpoint,
				ProxyType:          grpc_southbound.ProxyType(types.ProxyTypeMap[types.ProxyType(proxyType)]),
			})
		}

		hw_config_query := `
		SELECT
			id,
			device,
			ip_cidr::text
		FROM hardware_configs
		WHERE node_serial = $1 AND version_set_id = $2::uuid`

		rows, err = tx.Query(ctx, hw_config_query, SerialNumber, VersionSet)
		if err != nil {
			log.Error().Err(err).Msg("Failed to query hardware configs")
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var hwconfigID int32
			var hwconfigDevice string
			var hwconfigIPCIDR string
			err := rows.Scan(&hwconfigID, &hwconfigDevice, &hwconfigIPCIDR)
			if err != nil {
				log.Error().Err(err).Msg("Failed to scan hardware config")
				return err
			}
			node.HardwareConfig = append(node.HardwareConfig, &grpc_southbound.HardwareConfig{
				Id:     hwconfigID,
				Device: hwconfigDevice,
				IpCidr: hwconfigIPCIDR,
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

// GetVersionFleetUpdate retrieves all nodes for a specific version set
/* MUST BE TESTED */
func (s *StateManager) GetVersionFleetUpdate(ctx context.Context, versionSetId string) (*grpc_control_plane.FleetUpdate, error) {
	var nodes []*grpc_control_plane.NodeUpdateItem

	// Get all nodes for this version set
	query := `
		SELECT DISTINCT serial_number 
		FROM nodes 
		WHERE version_set_id = $1::uuid
		`

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, versionSetId)
		if err != nil {
			return fmt.Errorf("failed to query nodes: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var serialNumber string
			if err := rows.Scan(&serialNumber); err != nil {
				return fmt.Errorf("failed to scan node: %w", err)
			}

			// Get full node update for each node
			nodeUpdate, err := s.NodeUpdate(serialNumber, versionSetId, ctx)
			if err != nil {
				return fmt.Errorf("failed to get node update for %s: %w", serialNumber, err)
			}
			if nodeUpdate != nil {
				nodes = append(nodes, nodeUpdate)
			}
		}
		return rows.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get version fleet update: %w", err)
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	return &grpc_control_plane.FleetUpdate{
		NodeUpdateItems: nodes,
	}, nil
}

// GetGroupFleetUpdate retrieves all nodes for a specific group in a version set
/* MUST BE TESTED */
func (s *StateManager) GetGroupFleetUpdate(ctx context.Context, groupName string, versionSetId string) (*grpc_control_plane.FleetUpdate, error) {
	var nodes []*grpc_control_plane.NodeUpdateItem

	// Get all nodes in this group
	query := `
		SELECT DISTINCT p.node_serial
		FROM proxies p
		JOIN groups g ON p.group_name = g.name AND p.version_set_id = g.version_set_id
		WHERE p.group_name = $1 
		AND p.version_set_id = $2::uuid
		AND p.disabled_at IS NULL`

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, groupName, versionSetId)
		if err != nil {
			return fmt.Errorf("failed to query nodes: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var serialNumber string
			if err := rows.Scan(&serialNumber); err != nil {
				return fmt.Errorf("failed to scan node: %w", err)
			}

			// Get full node update for each node
			nodeUpdate, err := s.NodeUpdate(serialNumber, versionSetId, ctx)
			if err != nil {
				return fmt.Errorf("failed to get node update for %s: %w", serialNumber, err)
			}
			if nodeUpdate != nil {
				nodes = append(nodes, nodeUpdate)
			}
		}
		return rows.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get group fleet update: %w", err)
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	return &grpc_control_plane.FleetUpdate{
		NodeUpdateItems: nodes,
	}, nil
}

// GetFleetUpdateOptimized retrieves all nodes and their configurations in a single query
// If groupName is empty, it performs a version update, otherwise a group update
/* MUST BE TESTED */
func (s *StateManager) GetFleetUpdateOptimized(ctx context.Context, versionSetId string, groupName string) (*grpc_control_plane.FleetUpdate, error) {
	var query string
	var args []any

	query = `
		WITH target_nodes AS (
			SELECT DISTINCT serial_number
			FROM nodes
			WHERE version_set_id = $1::uuid
			AND now()- last_seen < interval '2 minute'
		)
		SELECT
			-- Node information
			n.serial_number,
			n.network_index,
			n.locality,
			n.version_set_id::text,

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
			p.name AS proxy_name,
			p.state AS proxy_state,
			p.proxy_type,
			p.server_endpoint_addr,
			p.client_endpoint_addr,

			-- Hardware Configurations
			hc.id AS hwconfig_id,
			hc.device AS hwconfig_device,
			hc.ip_cidr::text AS hwconfig_ip_cidr
		FROM target_nodes tn
		JOIN nodes n ON n.serial_number = tn.serial_number
		LEFT JOIN hardware_configs hc ON n.serial_number = hc.node_serial AND n.version_set_id::uuid = hc.version_set_id
		LEFT JOIN proxies p ON p.node_serial = n.serial_number AND p.version_set_id = n.version_set_id
		LEFT JOIN groups g ON p.group_name = g.name AND g.version_set_id = n.version_set_id
		LEFT JOIN endpoint_configs ec1 ON g.endpoint_config_name = ec1.name AND g.version_set_id = ec1.version_set_id
		LEFT JOIN endpoint_configs ec2 ON g.legacy_config_name = ec2.name AND g.version_set_id = ec2.version_set_id
		WHERE n.version_set_id = $1::uuid
		ORDER BY n.serial_number, g.name, p.name`

	args = []any{versionSetId}
	nodeMap := make(map[string]*grpc_control_plane.NodeUpdateItem)
	var nodes []*grpc_control_plane.NodeUpdateItem

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to query nodes: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				serialNumber   string
				networkIndex   int32
				locality       string
				versionSetId   string
				groupName      string
				groupLogLevel  int32
				endpointConfig types.EndpointConfig
				legacyConfig   types.EndpointConfig
				proxyName      string
				proxyState     bool
				proxyType      string
				serverEndpoint string
				clientEndpoint string
				hwconfigID     int32
				hwconfigDevice string
				hwconfigIPCIDR string
			)

			err := rows.Scan(
				&serialNumber, &networkIndex, &locality, &versionSetId,
				&groupName, &groupLogLevel,
				&endpointConfig.Name, &endpointConfig.MutualAuth, &endpointConfig.NoEncryption,
				&endpointConfig.ASLKeyExchangeMethod, &endpointConfig.Cipher,
				&legacyConfig.Name, &legacyConfig.MutualAuth, &legacyConfig.NoEncryption,
				&legacyConfig.ASLKeyExchangeMethod, &legacyConfig.Cipher,
				&proxyName, &proxyState, &proxyType, &serverEndpoint, &clientEndpoint,
				&hwconfigID, &hwconfigDevice, &hwconfigIPCIDR,
			)
			if err != nil {
				return fmt.Errorf("failed to scan row: %w", err)
			}

			// Get or create node
			node, exists := nodeMap[serialNumber]
			if !exists {
				node = &grpc_control_plane.NodeUpdateItem{
					SerialNumber:     serialNumber,
					NetworkIndex:     networkIndex,
					Locality:         locality,
					VersionSetId:     versionSetId,
					GroupProxyUpdate: []*grpc_control_plane.GroupProxyUpdate{},
				}
				nodeMap[serialNumber] = node
				nodes = append(nodes, node)
			}

			// Skip if no group info
			if groupName == "" {
				continue
			}

			// Find or create group update
			var groupUpdate *grpc_control_plane.GroupProxyUpdate
			for _, g := range node.GroupProxyUpdate {
				if g.GroupName == groupName {
					groupUpdate = g
					break
				}
			}
			if groupUpdate == nil {
				groupUpdate = &grpc_control_plane.GroupProxyUpdate{
					GroupName:     groupName,
					GroupLogLevel: groupLogLevel,
					EndpointConfig: &grpc_southbound.EndpointConfig{
						Name:                 endpointConfig.Name,
						MutualAuth:           endpointConfig.MutualAuth,
						NoEncryption:         endpointConfig.NoEncryption,
						AslKeyExchangeMethod: endpointConfig.ASLKeyExchangeMethod,
						Cipher:               endpointConfig.Cipher,
					},
					LegacyConfig: &grpc_southbound.EndpointConfig{
						Name:                 legacyConfig.Name,
						MutualAuth:           legacyConfig.MutualAuth,
						NoEncryption:         legacyConfig.NoEncryption,
						AslKeyExchangeMethod: legacyConfig.ASLKeyExchangeMethod,
						Cipher:               legacyConfig.Cipher,
					},
					Proxies: []*grpc_control_plane.UpdateProxy{},
				}
				node.GroupProxyUpdate = append(node.GroupProxyUpdate, groupUpdate)
			}

			// Add proxy if it exists
			if proxyName != "" {
				proxy := &grpc_control_plane.UpdateProxy{
					Name:               proxyName,
					ServerEndpointAddr: serverEndpoint,
					ClientEndpointAddr: clientEndpoint,
					ProxyType:          grpc_southbound.ProxyType(types.ProxyTypeMap[types.ProxyType(proxyType)]),
				}
				groupUpdate.Proxies = append(groupUpdate.Proxies, proxy)
			}
		}

		return rows.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get fleet update: %w", err)
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	return &grpc_control_plane.FleetUpdate{
		NodeUpdateItems: nodes,
	}, nil
}

func (s *StateManager) GetGroupUpdateOptimized(ctx context.Context, versionSetId string, groupName string) (*grpc_control_plane.FleetUpdate, error) {
	var query string
	var args []any

	query = `
		WITH target_nodes AS (
			SELECT DISTINCT p.node_serial
			FROM proxies p
			WHERE p.group_name = $1 
			AND p.version_set_id = $2::uuid
			AND now()- last_seen < interval '2 minute'
		)
		SELECT
			-- Node information
			n.serial_number,
			n.network_index,
			n.locality,
			n.version_set_id::text,
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
			p.name AS proxy_name,
			p.state AS proxy_state,
			p.proxy_type,
			p.server_endpoint_addr,
			p.client_endpoint_addr
		FROM target_nodes tn
		JOIN nodes n ON n.serial_number = tn.node_serial
		LEFT JOIN proxies p ON p.node_serial = n.serial_number AND p.version_set_id = n.version_set_id
		LEFT JOIN groups g ON p.group_name = g.name AND g.version_set_id = n.version_set_id
		LEFT JOIN endpoint_configs ec1 ON g.endpoint_config_name = ec1.name AND g.version_set_id = ec1.version_set_id
		LEFT JOIN endpoint_configs ec2 ON g.legacy_config_name = ec2.name AND g.version_set_id = ec2.version_set_id
		WHERE n.version_set_id = $2::uuid
		ORDER BY n.serial_number, g.name, p.name`
	args = []any{groupName, versionSetId}

	nodeMap := make(map[string]*grpc_control_plane.NodeUpdateItem)
	var nodes []*grpc_control_plane.NodeUpdateItem

	err := s.ExecuteInTransaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to query nodes: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				serialNumber   string
				networkIndex   int32
				locality       string
				versionSetId   string
				groupName      string
				groupLogLevel  int32
				endpointConfig types.EndpointConfig
				legacyConfig   types.EndpointConfig
				proxyName      string
				proxyState     bool
				proxyType      string
				serverEndpoint string
				clientEndpoint string
			)

			err := rows.Scan(
				&serialNumber, &networkIndex, &locality, &versionSetId,
				&groupName, &groupLogLevel,
				&endpointConfig.Name, &endpointConfig.MutualAuth, &endpointConfig.NoEncryption,
				&endpointConfig.ASLKeyExchangeMethod, &endpointConfig.Cipher,
				&legacyConfig.Name, &legacyConfig.MutualAuth, &legacyConfig.NoEncryption,
				&legacyConfig.ASLKeyExchangeMethod, &legacyConfig.Cipher,
				&proxyName, &proxyState, &proxyType, &serverEndpoint, &clientEndpoint,
			)
			if err != nil {
				return fmt.Errorf("failed to scan row: %w", err)
			}

			// Get or create node
			node, exists := nodeMap[serialNumber]
			if !exists {
				node = &grpc_control_plane.NodeUpdateItem{
					SerialNumber:     serialNumber,
					NetworkIndex:     networkIndex,
					Locality:         locality,
					VersionSetId:     versionSetId,
					GroupProxyUpdate: []*grpc_control_plane.GroupProxyUpdate{},
				}
				nodeMap[serialNumber] = node
				nodes = append(nodes, node)
			}

			// Skip if no group info
			if groupName == "" {
				continue
			}

			// Find or create group update
			var groupUpdate *grpc_control_plane.GroupProxyUpdate
			for _, g := range node.GroupProxyUpdate {
				if g.GroupName == groupName {
					groupUpdate = g
					break
				}
			}
			if groupUpdate == nil {
				groupUpdate = &grpc_control_plane.GroupProxyUpdate{
					GroupName:     groupName,
					GroupLogLevel: groupLogLevel,
					EndpointConfig: &grpc_southbound.EndpointConfig{
						Name:                 endpointConfig.Name,
						MutualAuth:           endpointConfig.MutualAuth,
						NoEncryption:         endpointConfig.NoEncryption,
						AslKeyExchangeMethod: endpointConfig.ASLKeyExchangeMethod,
						Cipher:               endpointConfig.Cipher,
					},
					LegacyConfig: &grpc_southbound.EndpointConfig{
						Name:                 legacyConfig.Name,
						MutualAuth:           legacyConfig.MutualAuth,
						NoEncryption:         legacyConfig.NoEncryption,
						AslKeyExchangeMethod: legacyConfig.ASLKeyExchangeMethod,
						Cipher:               legacyConfig.Cipher,
					},
					Proxies: []*grpc_control_plane.UpdateProxy{},
				}
				node.GroupProxyUpdate = append(node.GroupProxyUpdate, groupUpdate)
			}

			// Add proxy if it exists
			if proxyName != "" {
				proxy := &grpc_control_plane.UpdateProxy{
					Name:               proxyName,
					ServerEndpointAddr: serverEndpoint,
					ClientEndpointAddr: clientEndpoint,
					ProxyType:          grpc_southbound.ProxyType(types.ProxyTypeMap[types.ProxyType(proxyType)]),
				}
				groupUpdate.Proxies = append(groupUpdate.Proxies, proxy)
			}
		}

		return rows.Err()
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get fleet update: %w", err)
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	return &grpc_control_plane.FleetUpdate{
		NodeUpdateItems: nodes,
	}, nil
}
