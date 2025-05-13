package db

import (
	"context"
	"fmt"
	"os"
	"time"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type StateManager struct {
	pool *pgxpool.Pool
}

// Config holds database configuration
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DatabaseName string
	SSLMode      string
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		DatabaseName: "postgres",
	}
}

// BuildConnectionString creates a PostgreSQL connection string
func (c Config) BuildConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DatabaseName, c.SSLMode,
	)
}

func NewStateManager(ctx context.Context) (*StateManager, error) {
	log.Trace().Msg("in function new Statemanager")
	dbConfig := DefaultConfig()

	// Override with environment variables if needed
	if envHost := os.Getenv("DB_HOST"); envHost != "" {
		dbConfig.Host = envHost
	}

	pool, err := SetupDatabase(ctx, dbConfig)
	if err != nil {
		log.Err(err).Msgf("Failed to setup database: %v", err)
		return nil, err
	}

	return &StateManager{pool: pool}, nil
}

// SetupDatabase initializes the database and runs migrations
func SetupDatabase(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	log.Trace().Msg("in function SetupDatabase")
	// First, try to connect to create the database if it doesn't exist
	adminConnStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s",
		config.Host, config.Port, config.User, config.Password,
	)
	log.Debug().Msgf("admin connection is: %s", adminConnStr)
	adminPool, err := pgxpool.Connect(ctx, adminConnStr)
	if err != nil {
		log.Err(err).Msg("")
		log.Err(err).Msg("failed to connect to PostgreSQL")
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer adminPool.Close()

	// Connect to the specific database
	poolConfig, err := pgxpool.ParseConfig(config.BuildConnectionString())
	if err != nil {
		log.Err(err).Msgf("failed to parse connection string")
		return nil, err
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}

	// Set connection pool settings
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Create connection pool
	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		log.Err(err).Msg("failed to create connection pool")
		return nil, err
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Err(err).Msg("failed to ping database")
		return nil, err
	}

	return pool, nil
}

// InitializeSchema creates all necessary database tables and types
func (sm *StateManager) InitializeSchema() error {
	log.Debug().Msg("Initializing database schema")

	return sm.ExecuteInTransaction(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), schemaSQL)
		if err != nil {
			return fmt.Errorf("failed to initialize schema: %w", err)
		}
		return nil
	})
}

// ResetDatabase drops all tables and recreates them
func (sm *StateManager) ResetDatabase() error {
	log.Debug().Msg("Resetting database")

	// Drop all tables in reverse order of dependencies
	dropSQL := `
    drop trigger if exists trigger_update_node_disabled_status on nodes;
	drop function if exists trg_update_node_disabled_status();
	drop table if exists change_log cascade;
	drop table if exists version_sets cascade;
	drop table if exists version_transitions cascade;
	drop type if exists version_state cascade;
	drop type if exists version_transition_status cascade;
	drop type if exists transaction_type cascade;
	drop type if exists transaction_state cascade;
	drop table if exists hardware_configs cascade;
	drop table if exists proxies cascade;
	drop table if exists endpoint_configs cascade;
	drop table if exists groups cascade;
	drop table if exists nodes cascade;
	drop table if exists enroll cascade;
	drop table if exists transactions cascade;
	drop function if exists handle_transaction_rollback() cascade;
	drop table if exists transaction_log cascade;
	drop function if exists ensure_single_pending_transaction() cascade;
	drop function if exists create_new_pending_transaction() cascade;
	drop function if exists log_changes() cascade;
	drop function if exists process_rollback() cascade;
	drop function if exists complete_transaction() cascade;
	drop function if exists rollback_transaction() cascade;
	drop type if exists transaction_status cascade;
	drop type if exists proxy_type cascade;
	drop type if exists asl_key_exchange_method cascade;
	drop type if exists operation_type cascade;
	`
	return sm.ExecuteInTransaction(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), dropSQL)
		if err != nil {
			log.Err(err).Msg("failed to drop tables")
			return err
		}
		return nil
	})
}
