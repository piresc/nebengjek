package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PostgresClient represents a PostgreSQL database client
type PostgresClient struct {
	pool *pgxpool.Pool
}

// NewPostgresClient creates a new PostgreSQL client
func NewPostgresClient(config models.DatabaseConfig) (*PostgresClient, error) {
	// Build connection string
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.SSLMode,
	)

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Set max connections
	if config.MaxConns > 0 {
		poolConfig.MaxConns = int32(config.MaxConns)
	}

	// Set idle connections
	if config.IdleConns > 0 {
		poolConfig.MinConns = int32(config.IdleConns)
	}

	// Set connection lifetime, health check, etc.
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresClient{pool: pool}, nil
}

// GetPool returns the underlying connection pool
func (p *PostgresClient) GetPool() *pgxpool.Pool {
	return p.pool
}

// Close closes the database connection pool
func (p *PostgresClient) Close() {
	p.pool.Close()
}
