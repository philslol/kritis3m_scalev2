package db

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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
		node := &Node{
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
		node = &Node{
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
		node := &Node{
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
