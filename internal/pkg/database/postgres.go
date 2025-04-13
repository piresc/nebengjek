package database

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Standard library bindings for pgx
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PostgresClient represents a PostgreSQL database client
type PostgresClient struct {
	db *sqlx.DB
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

	// Create connection with sqlx
	db, err := sqlx.Connect("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Configure connection pool
	if config.MaxConns > 0 {
		db.SetMaxOpenConns(config.MaxConns)
	}

	if config.IdleConns > 0 {
		db.SetMaxIdleConns(config.IdleConns)
	}

	// Set connection lifetime
	db.SetConnMaxLifetime(1 * time.Hour)

	// Verify connection with context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresClient{db: db}, nil
}

// GetDB returns the underlying sqlx DB instance
func (p *PostgresClient) GetDB() *sqlx.DB {
	return p.db
}

// Close closes the database connection
func (p *PostgresClient) Close() {
	p.db.Close()
}
