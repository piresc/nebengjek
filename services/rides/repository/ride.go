package repository

import (
	"context"
	"fmt"
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

// AddBillingEntry adds a new entry to the billing ledger
func (r *RideRepo) AddBillingEntry(ctx context.Context, entry *models.BillingLedger) error {
	query := `
		INSERT INTO billing_ledger (
			entry_id, ride_id, distance, cost, created_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`

	if entry.EntryID == uuid.Nil {
		entry.EntryID = uuid.New()
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		entry.EntryID,
		entry.RideID,
		entry.Distance,
		entry.Cost,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to insert billing entry: %w", err)
	}

	return nil
}

// UpdateTotalCost updates the total cost of a ride
func (r *RideRepo) UpdateTotalCost(ctx context.Context, rideID string, additionalCost int) error {
	query := `
		UPDATE rides 
		SET total_cost = total_cost + $1,
			updated_at = NOW()
		WHERE ride_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, additionalCost, rideID)
	if err != nil {
		return fmt.Errorf("failed to update ride total cost: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("ride not found: %s", rideID)
	}

	return nil
}

// GetRide gets a ride by ID
func (r *RideRepo) GetRide(ctx context.Context, rideID string) (*models.Ride, error) {
	var ride models.Ride

	query := `
		SELECT ride_id, driver_id, customer_id, status, total_cost, created_at, updated_at
		FROM rides
		WHERE ride_id = $1
	`

	err := r.db.GetContext(ctx, &ride, query, rideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	return &ride, nil
}

// CompleteRide marks a ride as completed
func (r *RideRepo) CompleteRide(ctx context.Context, ride *models.Ride) error {
	query := `
		UPDATE rides 
		SET status = 'completed',
			updated_at = NOW()
		WHERE ride_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, ride.RideID)
	if err != nil {
		return fmt.Errorf("failed to complete ride: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("ride not found: %s", ride.RideID)
	}

	return nil
}

// GetBillingLedgerSum gets the sum of all costs in the billing ledger for a ride
func (r *RideRepo) GetBillingLedgerSum(ctx context.Context, rideID string) (int, error) {
	query := `
		SELECT SUM(cost) 
		FROM billing_ledger 
		WHERE ride_id = $1
	`

	var totalCost int
	err := r.db.GetContext(ctx, &totalCost, query, rideID)
	if err != nil {
		return 0, fmt.Errorf("failed to get billing ledger sum: %w", err)
	}

	return totalCost, nil
}

// CreatePayment creates a payment record for a ride
func (r *RideRepo) CreatePayment(ctx context.Context, payment *models.Payment) error {
	query := `
		INSERT INTO payments (
			payment_id, ride_id, adjusted_cost, admin_fee, driver_payout, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	if payment.PaymentID == uuid.Nil {
		payment.PaymentID = uuid.New()
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		payment.PaymentID,
		payment.RideID,
		payment.AdjustedCost,
		payment.AdminFee,
		payment.DriverPayout,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}
