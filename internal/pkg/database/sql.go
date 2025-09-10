package database

import (
	"context"
	"database/sql"
	"strings"

	"github.com/piresc/nebengjek/internal/pkg/tracing"
)

// TracedDB wraps a SQL database with automatic tracing
type TracedDB struct {
	*sql.DB
	tracer tracing.Tracer
}

// NewTracedDB creates a new traced SQL database
func NewTracedDB(db *sql.DB, tracer tracing.Tracer) *TracedDB {
	return &TracedDB{DB: db, tracer: tracer}
}

// QueryContext executes a query with automatic tracing
func (db *TracedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !db.tracer.IsEnabled() {
		return db.DB.QueryContext(ctx, query, args...)
	}

	operation, table := parseSQLQuery(query)
	ctx, done := db.tracer.StartDatabase(ctx, operation, table)
	defer done()

	return db.DB.QueryContext(ctx, query, args...)
}

// ExecContext executes a statement with automatic tracing
func (db *TracedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !db.tracer.IsEnabled() {
		return db.DB.ExecContext(ctx, query, args...)
	}

	operation, table := parseSQLQuery(query)
	ctx, done := db.tracer.StartDatabase(ctx, operation, table)
	defer done()

	return db.DB.ExecContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns at most one row with automatic tracing
func (db *TracedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if !db.tracer.IsEnabled() {
		return db.DB.QueryRowContext(ctx, query, args...)
	}

	operation, table := parseSQLQuery(query)
	ctx, done := db.tracer.StartDatabase(ctx, operation, table)
	defer done()

	return db.DB.QueryRowContext(ctx, query, args...)
}

// BeginTx begins a transaction with automatic tracing
func (db *TracedDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if !db.tracer.IsEnabled() {
		return db.DB.BeginTx(ctx, opts)
	}

	ctx, done := db.tracer.StartDatabase(ctx, "BEGIN", "transaction")
	defer done()

	return db.DB.BeginTx(ctx, opts)
}

// parseSQLQuery extracts operation and table from SQL query
func parseSQLQuery(query string) (operation, table string) {
	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	// Extract operation (SELECT, INSERT, UPDATE, DELETE, etc.)
	parts := strings.Fields(upperQuery)
	if len(parts) > 0 {
		operation = parts[0]
	}

	// Extract table name (simplified extraction)
	switch operation {
	case "SELECT":
		if fromIdx := strings.Index(upperQuery, " FROM "); fromIdx != -1 {
			remaining := query[fromIdx+6:]
			tableEnd := strings.IndexAny(remaining, " \n,;")
			if tableEnd != -1 {
				table = remaining[:tableEnd]
			} else {
				table = remaining
			}
		}
	case "INSERT":
		if intoIdx := strings.Index(upperQuery, " INTO "); intoIdx != -1 {
			remaining := query[intoIdx+6:]
			tableEnd := strings.IndexAny(remaining, " \n,(")
			if tableEnd != -1 {
				table = remaining[:tableEnd]
			} else {
				table = remaining
			}
		}
	case "UPDATE":
		if len(parts) > 1 {
			table = parts[1]
		}
	case "DELETE":
		if fromIdx := strings.Index(upperQuery, " FROM "); fromIdx != -1 {
			remaining := query[fromIdx+6:]
			tableEnd := strings.IndexAny(remaining, " \n,;")
			if tableEnd != -1 {
				table = remaining[:tableEnd]
			} else {
				table = remaining
			}
		}
	}

	return operation, table
}