package db

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestTransactionMechanism(t *testing.T) {

	// Test case 1: Successful transaction
	t.Run("Successful Transaction - Create Node and Rollback Update", func(t *testing.T) {
		sm, err := NewStateManager()
		assert.NoError(t, err, "Failed to initialize StateManager")

		ctx := context.Background()

		// Setup database schema
		_, err = sm.pool.Exec(ctx, schemaSQL)

		_, err = sm.pool.Exec(ctx, functionsSQL)
		assert.NoError(t, err, "Failed to execute functions SQL")

		// Create Node instance
		lastSeen := pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		}
		bla := rand.Int32()
		serial_number := fmt.Sprintf("test-%d", bla)
		node := &types.Node{
			SerialNumber: serial_number,
			NetworkIndex: 1,
			Locality:     "TestLocation",
			LastSeen:     lastSeen,
			CreatedBy:    "test_user",
		}

		// Create node
		err = sm.CreateNode(ctx, node)
		assert.NoError(t, err)

		err = sm.CompleteTransaction(ctx)
		assert.NoError(t, err)

		// Create a second node for update testing
		lastSeen = pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		}
		bla = rand.Int32()
		serial_number = fmt.Sprintf("test-%d", bla)
		node = &types.Node{
			SerialNumber: serial_number,
			NetworkIndex: 1,
			Locality:     "TestLocation",
			LastSeen:     lastSeen,
			CreatedBy:    "test_user",
		}

		err = sm.CreateNode(ctx, node)
		err = sm.CompleteTransaction(ctx)
		assert.NoError(t, err)

		node.Locality = "update location"
		err = sm.UpdateNode(ctx, node)
		assert.NoError(t, err)

		// Rollback transaction and verify rollback
		err = sm.rollbackTransaction(ctx)
		assert.NoError(t, err)

		node, err = sm.GetNode(ctx, node.ID)
		assert.NoError(t, err)
		assert.Equal(t, "TestLocation", node.Locality)
	})

	// Test case 2: Successful transaction - Delete Node and Rollback
	t.Run("Successful Transaction - Delete Node and Rollback", func(t *testing.T) {
		sm, err := NewStateManager()
		assert.NoError(t, err, "Failed to initialize StateManager")

		ctx := context.Background()

		_, err = sm.pool.Exec(ctx, functionsSQL)
		assert.NoError(t, err, "Failed to execute functions SQL")

		// Create Node instance
		lastSeen := pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		}
		bla := rand.Int32()
		serial_number := fmt.Sprintf("test-%d", bla)
		node := &types.Node{
			SerialNumber: serial_number,
			NetworkIndex: 1,
			Locality:     "TestLocation",
			LastSeen:     lastSeen,
			CreatedBy:    "test_user",
		}

		// Create node
		err = sm.CreateNode(ctx, node)
		assert.NoError(t, err)

		err = sm.CompleteTransaction(ctx)
		assert.NoError(t, err)

		// Delete node
		err = sm.DeleteNode(ctx, node.ID)
		assert.NoError(t, err)

		// Rollback transaction and verify the node exists again
		err = sm.rollbackTransaction(ctx)
		assert.NoError(t, err)

		node, err = sm.GetNode(ctx, node.ID)
		assert.NoError(t, err)
		assert.NotNil(t, node)
	})
}

func TestNodeCRUD(t *testing.T) {

	sm, err := NewStateManager()
	ctx := context.Background()
	sm.CompleteTransaction(ctx)
	err = sm.rollbackTransaction(ctx)
	if err != nil {
		log.Err(err)
	}
	assert.NoError(t, err)

	// Test Create
	node := &types.Node{
		SerialNumber: "TEST001",
		NetworkIndex: 1,
		Locality:     "test-locality",
		LastSeen:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CreatedBy:    "test-user",
	}

	err = sm.CreateNode(ctx, node)
	assert.NoError(t, err)
	assert.NotZero(t, node.ID)
	assert.NotZero(t, node.CreatedAt)
	assert.NotZero(t, node.UpdatedAt)

	// Test Get
	fetchedNode, err := sm.GetNode(ctx, node.ID)
	assert.NoError(t, err)
	assert.Equal(t, node.SerialNumber, fetchedNode.SerialNumber)
	assert.Equal(t, node.NetworkIndex, fetchedNode.NetworkIndex)
	assert.Equal(t, node.Locality, fetchedNode.Locality)

	// Test Update
	node.Locality = "updated-locality"
	err = sm.UpdateNode(ctx, node)
	assert.NoError(t, err)
	assert.NotEqual(t, node.CreatedAt, node.UpdatedAt)

	// Test Delete
	err = sm.DeleteNode(ctx, node.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = sm.GetNode(ctx, node.ID)
	assert.Error(t, err)
}

func TestEndpointConfigCRUD(t *testing.T) {
	ctx := context.Background()

	// Setup database schema
	sm, err := NewStateManager()

	_, err = sm.pool.Exec(ctx, schemaSQL)
	if err != nil {
		log.Err(err)
	}

	_, err = sm.pool.Exec(ctx, functionsSQL)

	config := &types.EndpointConfig{
		Name:                 "test-config",
		MutualAuth:           true,
		NoEncryption:         false,
		ASLKeyExchangeMethod: "ASL_KEX_DEFAULT",
		Cipher:               "test-cipher",
		Status:               "pending",
		Version:              1,
		CreatedBy:            "test-user",
	}

	err = sm.CreateEndpointConfig(ctx, config)
	assert.NoError(t, err)
	assert.NotZero(t, config.ID)

	fetchedConfig, err := sm.GetEndpointConfig(ctx, config.ID)
	assert.NoError(t, err)
	assert.Equal(t, config.Name, fetchedConfig.Name)
	assert.Equal(t, config.MutualAuth, fetchedConfig.MutualAuth)

	config.Name = "updated-config"
	err = sm.UpdateEndpointConfig(ctx, config)
	assert.NoError(t, err)

	err = sm.DeleteEndpointConfig(ctx, config.ID)
	assert.NoError(t, err)
	err = sm.rollbackTransaction(ctx)
	if err != nil {
		log.Err(err)
	}

}

func TestGroupCRUD(t *testing.T) {
	ctx := context.Background()

	sm, err := NewStateManager()

	group := &types.Group{
		Name:      "test-group",
		LogLevel:  1,
		Status:    "pending",
		Version:   1,
		CreatedBy: "test-user",
	}

	err = sm.CreateGroup(ctx, group)
	assert.NoError(t, err)
	assert.NotZero(t, group.ID)

	fetchedGroup, err := sm.GetGroup(ctx, group.ID)
	assert.NoError(t, err)
	assert.Equal(t, group.Name, fetchedGroup.Name)
	assert.Equal(t, group.LogLevel, fetchedGroup.LogLevel)

	group.Name = "updated-group"
	err = sm.UpdateGroup(ctx, group)
	assert.NoError(t, err)

}

func TestHardwareConfigCRUD(t *testing.T) {
	ctx := context.Background()
	sm, err := NewStateManager()
	// Alternative way using String method
	ipcidr := pgtype.Inet{}
	err = ipcidr.Scan("192.168.1.0/24")

	if err != nil {
		log.Err(err)
	}
	config := &types.HardwareConfig{
		NodeID:    1,
		Device:    "test-device",
		IPCIDR:    ipcidr,
		Status:    "pending",
		Version:   1,
		CreatedBy: "test-user",
	}

	err = sm.CreateHardwareConfig(ctx, config)
	assert.NoError(t, err)
	assert.NotZero(t, config.ID)

	fetchedConfig, err := sm.GetHardwareConfig(ctx, config.ID)
	assert.NoError(t, err)
	assert.Equal(t, config.Device, fetchedConfig.Device)
	assert.Equal(t, config.IPCIDR, fetchedConfig.IPCIDR)

	config.Device = "updated-device"
	err = sm.UpdateHardwareConfig(ctx, config)
	assert.NoError(t, err)

	err = sm.DeleteHardwareConfig(ctx, config.ID)
	assert.NoError(t, err)
}

func TestProxyCRUD(t *testing.T) {
	ctx := context.Background()

	sm, err := NewStateManager()

	proxy := &types.Proxy{
		NodeID:             1,
		GroupID:            4,
		State:              true,
		ProxyType:          "FORWARD",
		ServerEndpointAddr: "server:8080",
		ClientEndpointAddr: "client:8080",
		Status:             "pending",
		Version:            1,
		CreatedBy:          "test-user",
	}

	err = sm.CreateProxy(ctx, proxy)
	assert.NoError(t, err)
	assert.NotZero(t, proxy.ID)

	fetchedProxy, err := sm.GetProxy(ctx, proxy.ID)
	assert.NoError(t, err)
	assert.Equal(t, proxy.ServerEndpointAddr, fetchedProxy.ServerEndpointAddr)
	assert.Equal(t, proxy.ClientEndpointAddr, fetchedProxy.ClientEndpointAddr)

	proxy.State = false
	err = sm.UpdateProxy(ctx, proxy)
	assert.NoError(t, err)

	err = sm.DeleteProxy(ctx, proxy.ID)
	assert.NoError(t, err)
}
