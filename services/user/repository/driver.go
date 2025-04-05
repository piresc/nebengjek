package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// Driver-specific methods for UserRepo implementation

// UpdateDriverLocation updates a driver's current location
func (r *UserRepo) UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error {
	// Ensure timestamp is set
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Insert new location
	_, err := r.db.Exec(ctx, `
		INSERT INTO driver_locations (user_id, latitude, longitude, address, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`, driverID, location.Latitude, location.Longitude, location.Address, location.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}

	return nil
}

// UpdateDriverAvailability updates a driver's availability status
func (r *UserRepo) UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error {
	result, err := r.db.Exec(ctx, `
		UPDATE drivers SET is_available = $1
		WHERE user_id = $2
	`, isAvailable, driverID)
	if err != nil {
		return fmt.Errorf("failed to update driver availability: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver not found")
	}

	return nil
}

// GetNearbyDrivers retrieves available drivers near a location within a radius (continued)
func (r *UserRepo) getNearbyDriversDocuments(ctx context.Context, driverID string) ([]string, error) {
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
func (r *UserRepo) VerifyDriver(ctx context.Context, driverID string) error {
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
