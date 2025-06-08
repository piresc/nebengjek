package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/logger"
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
			ride_id, match_id, driver_id, passenger_id, status, total_cost, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING ride_id
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		ride.RideID,
		ride.MatchID,
		ride.DriverID,
		ride.PassengerID,
		ride.Status,
		ride.TotalCost,
		ride.CreatedAt,
		ride.UpdatedAt,
	)

	if err != nil {
		logger.Error("Failed to create ride", logger.ErrorField(err))
		return nil, err
	}

	logger.Info("Created ride", logger.String("rideID", ride.RideID.String()))
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
	logger.Info("Getting ride from database",
		logger.String("ride_id", rideID))

	var ride models.Ride
	rideIDUUID, err := uuid.Parse(rideID)
	if err != nil {
		logger.Error("Invalid ride ID format",
			logger.String("ride_id", rideID),
			logger.ErrorField(err))
		return nil, fmt.Errorf("invalid ride ID format: %w", err)
	}

	query := `
		SELECT ride_id, match_id, driver_id, passenger_id, status, total_cost, created_at, updated_at
		FROM rides
		WHERE ride_id = $1
	`

	err = r.db.GetContext(ctx, &ride, query, rideIDUUID)
	if err != nil {
		logger.Error("Failed to get ride from database",
			logger.String("ride_id", rideID),
			logger.String("query", query),
			logger.ErrorField(err))
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	logger.Info("Successfully retrieved ride from database",
		logger.String("ride_id", rideID),
		logger.String("status", string(ride.Status)),
		logger.String("driver_id", ride.DriverID.String()),
		logger.String("passenger_id", ride.PassengerID.String()))

	return &ride, nil
}

// CompleteRide marks a ride as completed
func (r *RideRepo) CompleteRide(ctx context.Context, ride *models.Ride) error {
	query := `
		UPDATE rides 
		SET status = $1,
			updated_at = NOW()
		WHERE ride_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, models.RideStatusCompleted, ride.RideID)
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
			payment_id, ride_id, adjusted_cost, admin_fee, driver_payout, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
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
		payment.Status,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// UpdateRideStatus updates the status of a ride
func (r *RideRepo) UpdateRideStatus(ctx context.Context, rideID string, status models.RideStatus) error {
	logger.Info("Updating ride status",
		logger.String("ride_id", rideID),
		logger.String("new_status", string(status)))

	query := `
		UPDATE rides
		SET status = $1,
			updated_at = NOW()
		WHERE ride_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, rideID)
	if err != nil {
		logger.Error("Failed to update ride status in database",
			logger.String("ride_id", rideID),
			logger.String("new_status", string(status)),
			logger.ErrorField(err))
		return fmt.Errorf("failed to update ride status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Error("Failed to get affected rows after status update",
			logger.String("ride_id", rideID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		logger.Error("No rows affected - ride not found for status update",
			logger.String("ride_id", rideID),
			logger.String("new_status", string(status)))
		return fmt.Errorf("ride not found: %s", rideID)
	}

	logger.Info("Successfully updated ride status",
		logger.String("ride_id", rideID),
		logger.String("new_status", string(status)),
		logger.Int64("rows_affected", rows))

	return nil
}

// GetPaymentByRideID retrieves payment information for a specific ride
func (r *RideRepo) GetPaymentByRideID(ctx context.Context, rideID string) (*models.Payment, error) {
	var payment models.Payment
	rideIDUUID, err := uuid.Parse(rideID)
	if err != nil {
		return nil, fmt.Errorf("invalid ride ID format: %w", err)
	}

	query := `
		SELECT payment_id, ride_id, adjusted_cost, admin_fee, driver_payout, status, created_at
		FROM payments
		WHERE ride_id = $1
	`

	err = r.db.GetContext(ctx, &payment, query, rideIDUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment for ride %s: %w", rideID, err)
	}

	return &payment, nil
}

// UpdatePaymentStatus updates the status of a payment
func (r *RideRepo) UpdatePaymentStatus(ctx context.Context, paymentID string, status models.PaymentStatus) error {
	query := `
		UPDATE payments
		SET status = $1
		WHERE payment_id = $2
	`

	paymentUUID, err := uuid.Parse(paymentID)
	if err != nil {
		return fmt.Errorf("invalid payment ID format: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, status, paymentUUID)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}
