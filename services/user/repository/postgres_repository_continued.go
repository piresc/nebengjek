package repository

import (
	"context"
	"fmt"
)

// Continuation of PostgresUserRepository implementation

// GetNearbyDrivers retrieves available drivers near a location within a radius (continued)
func (r *PostgresUserRepository) getNearbyDriversDocuments(ctx context.Context, driverID string) ([]string, error) {
	// Get driver documents
	docRows, err := r.db.Query(ctx, `
		SELECT document_url FROM driver_documents
		WHERE user_id = $1
	`, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver documents: %w", err)
	}
	defer docRows.Close()

	var documents []string
	for docRows.Next() {
		var doc string
		if err := docRows.Scan(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan driver document: %w", err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// VerifyDriver marks a driver as verified
func (r *PostgresUserRepository) VerifyDriver(ctx context.Context, driverID string) error {
	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if driver exists
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users u
			JOIN drivers d ON u.id = d.user_id
			WHERE u.id = $1 AND u.role = 'driver'
		)
	`, driverID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check driver existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("driver not found")
	}

	// Update driver verification status
	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET verified = true, verified_at = NOW()
		WHERE user_id = $1
	`, driverID)
	if err != nil {
		return fmt.Errorf("failed to verify driver: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
