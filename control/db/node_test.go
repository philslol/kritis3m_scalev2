package db

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

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
		err = sm.CreateNode(ctx, node)
		assert.NoError(t, err)

		err = sm.CompleteTransaction(ctx)
		assert.NoError(t, err)

		lastSeen = pgtype.Timestamptz{
			Time:  time.Now(), // Set current time
			Valid: true,       // Mark the timestamp as valid
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
		node.Locality = "update location"
		sm.UpdateNode(ctx, node)

		//check status of rollback
		err = sm.rollbackTransaction(ctx)
		assert.NoError(t, err)

		//we assume, that this node is not available and returns an error
		_, err := sm.GetNode(ctx, node.ID)
		assert.Error(t, err)

	})

}
