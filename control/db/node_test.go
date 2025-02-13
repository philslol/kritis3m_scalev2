package db

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestTransactionMechanism(t *testing.T) {

	// Setup database connection
	sm, err := NewStateManager()
	if err != nil {
		log.Err(err)
	}

	// Test case 1: Successful transaction
	t.Run("Successful Transaction", func(t *testing.T) {
		ctx := context.Background()
		// Start transaction
		transactionID, err := sm.StartTransaction(ctx, "test_user", "Test transaction")
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, transactionID)

		// Create test node
		// Create Node instance
		lastSeen := pgtype.Timestamptz{
			Time:  time.Now(), // Set current time
			Valid: true,       // Mark the timestamp as valid
		}

		bla := rand.Int32()
		serial_number := fmt.Sprintf("test-%d", bla)
		node := &Node{
			SerialNumber: serial_number,
			NetworkIndex: 1,
			Locality:     "TestLocation",
			LastSeen:     lastSeen,
			CreatedBy:    "test_user",
		}

		// Create node
		err = sm.CreateNode(ctx, transactionID, node)
		assert.NoError(t, err)
		assert.NotEqual(t, 0, node.ID)

		// Update node
		node.Locality = "UpdatedLocation"
		err = sm.UpdateNode(ctx, transactionID, node)
		assert.NoError(t, err)

		// Apply changes
		err = sm.ApplyChanges(ctx, transactionID, "test_user")
		assert.NoError(t, err)

		// Verify changes
		updatedNode, err := sm.GetNode(ctx, node.ID)
		assert.NoError(t, err)
		assert.Equal(t, "UpdatedLocation", updatedNode.Locality)
	})

	// Test case 2: Failed transaction
	t.Run("Failed Transaction", func(t *testing.T) {
		ctx := context.Background()
		// Start transaction
		transactionID, err := sm.StartTransaction(ctx, "test_user", "Test failed transaction")
		assert.NoError(t, err)

		lastSeen := pgtype.Timestamptz{Time: time.Now(), Valid: true}

		bla := rand.Int32()
		serial_number := fmt.Sprintf("test-%d", bla)

		// Create test node
		node := &Node{
			SerialNumber: serial_number,
			NetworkIndex: 2,
			Locality:     "TestLocation2",
			LastSeen:     lastSeen,
			CreatedBy:    "test_user",
		}

		// Create node
		err = sm.CreateNode(ctx, transactionID, node)
		assert.NoError(t, err)

		// Force failure in ApplyChanges by manipulating random seed
		// Note: This relies on the 30% failure rate in ApplyNetworkChanges
		for i := 0; i < 10; i++ { // Try multiple times to ensure we hit a failure case
			err = sm.ApplyChanges(ctx, transactionID, "test_user")
			if err != nil {
				// Verify transaction was rolled back
				var status string
				err = sm.pool.QueryRow(ctx, "SELECT status FROM transactions WHERE id = $1", transactionID).Scan(&status)
				assert.NoError(t, err)
				assert.Equal(t, "rollback", status)
				return
			}
		}
		t.Error("Expected at least one transaction to fail")
	})
}
