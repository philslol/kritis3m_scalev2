package db

import (
	"context"
	"embed"
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

var migrations embed.FS

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
		SSLMode:      "disable",
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

	// Get database configuration
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
	// First, try to connect to create the database if it doesn't exist
	adminConnStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.SSLMode,
	)
	adminPool, err := pgxpool.Connect(ctx, adminConnStr)
	if err != nil {
		log.Err(err).Msgf("failed to connect to PostgreSQL: %w", err)
		return nil, err
	}
	defer adminPool.Close()

	// Create database if it doesn't exist
	_, err = adminPool.Exec(ctx, fmt.Sprintf(`
		CREATE DATABASE %s
		WITH 
		OWNER = %s
		ENCODING = 'UTF8'
		LC_COLLATE = 'en_US.utf8'
		LC_CTYPE = 'en_US.utf8'
		TEMPLATE template0;
	`, config.DatabaseName, config.User))

	if err != nil {
		// Ignore error if database already exists
		log.Printf("Note: database might already exist: %v", err)
	}

	// Connect to the specific database
	poolConfig, err := pgxpool.ParseConfig(config.BuildConnectionString())
	if err != nil {
		log.Err(err).Msgf("failed to parse connection string: %w", err)
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
		log.Err(err).Msgf("failed to create connection pool: %w", err)
		return nil, err
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Err(err).Msgf("failed to ping database: %w", err)
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
	DROP TABLE IF EXISTS proxies CASCADE;
	DROP TABLE IF EXISTS hardware_configs CASCADE;
	DROP TABLE IF EXISTS groups CASCADE;
	DROP TABLE IF EXISTS endpoint_configs CASCADE;
	DROP TABLE IF EXISTS nodes CASCADE;
	DROP TABLE IF EXISTS transaction_log CASCADE;
	DROP TABLE IF EXISTS transactions CASCADE;
	DROP TABLE IF EXISTS version_transitions CASCADE;
	DROP TABLE IF EXISTS version_sets CASCADE;
	
	DROP TYPE IF EXISTS proxy_type CASCADE;
	DROP TYPE IF EXISTS version_state CASCADE;
	DROP TYPE IF EXISTS version_transition_status CASCADE;
	DROP TYPE IF EXISTS transaction_type CASCADE;
	DROP TYPE IF EXISTS transaction_state CASCADE;
	DROP TYPE IF EXISTS operation_type CASCADE;
	DROP TYPE IF EXISTS asl_key_exchange_method CASCADE;
	`
	return sm.ExecuteInTransaction(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), dropSQL)
		if err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}
		return nil
	})
}
