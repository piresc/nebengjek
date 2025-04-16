package repository

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

type RideRepo struct {
	cfg *models.Config
	db  *sqlx.DB
}

func NewRideRepository(
	cfg *models.Config,
	db *sqlx.DB,
) *RideRepo {
	log.Println("Initializing ride repository")
	return &RideRepo{
		cfg: cfg,
		db:  db,
	}
}

// CreateRide creates a new ride in the database
func (r *RideRepo) CreateRide(ride *models.Ride) (*models.Ride, error) {
	ctx := context.Background()

	// Generate a new UUID if not provided
	if ride.RideID == uuid.Nil {
		ride.RideID = uuid.New()
	}

	// Set timestamps
	now := time.Now()
	ride.CreatedAt = now
	ride.UpdatedAt = now

	// Insert the ride into the database
	query := `
		INSERT INTO rides (
			ride_id, driver_id, customer_id, status, total_cost, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING ride_id
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		ride.RideID,
		ride.DriverID,
		ride.CustomerID,
		ride.Status,
		ride.TotalCost,
		ride.CreatedAt,
		ride.UpdatedAt,
	)

	if err != nil {
		log.Printf("Failed to create ride: %v", err)
		return nil, err
	}

	log.Printf("Created ride with ID: %s", ride.RideID)
	return ride, nil
}
