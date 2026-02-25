package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/models/ride"
	"github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/rides"
)

// RideUC implements the rides.RideUseCase interface
type rideUC struct {
	cfg       *core.Config
	ridesRepo rides.RideRepo
	ridesGW   rides.RideGW
}

// NewRideUC creates a new ride use case
func NewRideUC(
	cfg *core.Config,
	rideRepo rides.RideRepo,
	rideGW rides.RideGW,
) (rides.RideUC, error) {
	return &rideUC{
		cfg:       cfg,
		ridesRepo: rideRepo,
		ridesGW:   rideGW,
	}, nil
}

// CreateRide creates a new ride from a confirmed match
func (uc *rideUC) CreateRide(ctx context.Context, mp match.MatchProposal) error {
	// Parse UUIDs safely
	matchID, err := uuid.Parse(mp.ID)
	if err != nil {
		return fmt.Errorf("invalid match ID format: %w", err)
	}

	driverID, err := uuid.Parse(mp.DriverID)
	if err != nil {
		return fmt.Errorf("invalid driver ID format: %w", err)
	}

	passengerID, err := uuid.Parse(mp.PassengerID)
	if err != nil {
		return fmt.Errorf("invalid passenger ID format: %w", err)
	}

	// Create a new ride from the match proposal
	ride := &ride.Ride{
		MatchID:     matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ride.RideStatusDriverPickup, // Set initial status to driver pickup
		TotalCost:   0,                           // This will be calculated later
	}

	// Delegate to repository
	createdRide, err := uc.ridesRepo.CreateRide(ride)
	if err != nil {
		// Check if this is a duplicate match_id constraint violation
		if strings.Contains(err.Error(), "rides_match_id_unique") ||
			strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			logger.Warn("Ride already exists for this match - ignoring duplicate creation attempt",
				logger.String("match_id", mp.ID),
				logger.String("driver_id", mp.DriverID),
				logger.String("passenger_id", mp.PassengerID))
			// Return success since the ride already exists for this match
			return nil
		}

		logger.Error("Failed to create ride in database",
			logger.String("match_id", mp.ID),
			logger.String("driver_id", mp.DriverID),
			logger.String("passenger_id", mp.PassengerID),
			logger.ErrorField(err))
		return err
	}

	err = uc.ridesGW.PublishRidePickup(ctx, createdRide)
	if err != nil {
		logger.Error("Failed to publish ride pickup event to NATS",
			logger.String("ride_id", createdRide.RideID.String()),
			logger.String("driver_id", createdRide.DriverID.String()),
			logger.String("passenger_id", createdRide.PassengerID.String()),
			logger.ErrorField(err))
		return err
	}

	logger.Info("Successfully created ride and published pickup event",
		logger.String("ride_id", createdRide.RideID.String()))
	return nil
}

// ProcessBillingUpdate handles billing updates from location aggregates
func (uc *rideUC) ProcessBillingUpdate(ctx context.Context, rideID string, entry *ride.BillingLedger) error {

	// Get current ride to verify it exists and is active
	rideObj, err := uc.ridesRepo.GetRide(ctx, rideID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	if rideObj.Status != ride.RideStatusOngoing {
		return fmt.Errorf("cannot update billing for non-active ride")
	}

	// Parse ride ID to UUID
	rideUUID, err := uuid.Parse(rideID)
	if err != nil {
		return fmt.Errorf("invalid ride ID format: %w", err)
	}
	entry.RideID = rideUUID

	// Add billing entry
	if err := uc.ridesRepo.AddBillingEntry(ctx, entry); err != nil {
		return fmt.Errorf("failed to add billing entry: %w", err)
	}

	// Update total cost
	if err := uc.ridesRepo.UpdateTotalCost(ctx, rideID, entry.Cost); err != nil {
		logger.Warn("Failed to update total cost for ride",
			logger.String("ride_id", rideID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to update total cost: %w", err)
	}

	logger.Info("Updated billing for ride",
		logger.String("ride_id", rideID),
		logger.Int("cost", entry.Cost),
		logger.Float64("distance", entry.Distance))
	return nil
}

// StartRide updates a ride from driver_pickup to ongoing status
func (uc *rideUC) StartRide(ctx context.Context, req ride.RideStartRequest) (*ride.Ride, error) {
	// Get current ride to verify it exists and is in pickup state
	rideObj, err := uc.ridesRepo.GetRide(ctx, req.RideID)
	if err != nil {
		logger.Error("Failed to get ride for start request",
			logger.String("ride_id", req.RideID),
			logger.ErrorField(err))
		return &ride.Ride{}, fmt.Errorf("failed to get ride: %w", err)
	}

	if rideObj.Status != ride.RideStatusDriverPickup {
		logger.Error("Cannot start ride - invalid status",
			logger.String("ride_id", req.RideID),
			logger.String("current_status", string(rideObj.Status)),
			logger.String("required_status", string(ride.RideStatusDriverPickup)))
		err := fmt.Errorf("cannot start trip for ride not in driver_pickup state, current status: %s", rideObj.Status)
		return &ride.Ride{}, err
	}

	// Calculate distance using Haversine formula
	driverLoc := utils.GeoPoint{
		Latitude:  req.DriverLocation.Latitude,
		Longitude: req.DriverLocation.Longitude,
	}
	passLoc := utils.GeoPoint{
		Latitude:  req.PassengerLocation.Latitude,
		Longitude: req.PassengerLocation.Longitude,
	}

	// Verify driver is close to passenger (within configured distance)
	distanceKm := utils.CalculateDistance(driverLoc, passLoc)
	distanceMeters := distanceKm * 1000
	maxPickupDistance := uc.cfg.Rides.MaxPickupDistanceM

	// Check if driver is close enough to passenger
	if distanceMeters > maxPickupDistance {
		logger.Error("Driver too far from passenger",
			logger.String("ride_id", req.RideID),
			logger.Float64("distance_meters", distanceMeters),
			logger.Float64("max_allowed_meters", maxPickupDistance))
		err := fmt.Errorf("driver is too far from passenger (%.2f meters)", distanceMeters)
		return &ride.Ride{}, err
	}

	// Update ride status to ongoing
	rideObj.Status = ride.RideStatusOngoing
	if err := uc.ridesRepo.UpdateRideStatus(ctx, rideObj.RideID.String(), ride.RideStatusOngoing); err != nil {
		return &ride.Ride{}, fmt.Errorf("failed to update ride status to ongoing: %w", err)
	}

	logger.Info("Ride started - Driver picked up passenger",
		logger.String("ride_id", req.RideID))
	return rideObj, nil
}

// RideArrived handles when a ride arrives at the destination but before payment processing
func (uc *rideUC) RideArrived(ctx context.Context, req ride.RideArrivalReq) (*ride.PaymentRequest, error) {
	// Get current ride to verify it exists and is active
	rideObj, err := uc.ridesRepo.GetRide(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	if rideObj.Status != ride.RideStatusOngoing {
		err := fmt.Errorf("cannot process arrival for ride that is not ongoing")
		return nil, err
	}

	// Get total cost from billing ledger (to ensure accuracy)
	totalCost, err := uc.ridesRepo.GetBillingLedgerSum(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total cost: %w", err)
	}

	// Validate adjustment factor
	if req.AdjustmentFactor < 0 || req.AdjustmentFactor > 1.0 {
		req.AdjustmentFactor = 1.0 // Reset to 100% if invalid
	}

	// Calculate adjusted cost
	adjustedCost := int(float64(totalCost) * req.AdjustmentFactor)

	adminFeePercent := uc.cfg.Pricing.AdminFeePercent / 100.0 // Convert percentage to decimal
	adminFee := int(float64(adjustedCost) * adminFeePercent)
	driverPayout := adjustedCost - adminFee

	// Create payment record
	payment := &ride.Payment{
		PaymentID:    uuid.New(),
		RideID:       rideObj.RideID,
		AdjustedCost: adjustedCost,
		AdminFee:     adminFee,
		DriverPayout: driverPayout,
		Status:       ride.PaymentStatusPending,
		CreatedAt:    time.Now(),
	}

	// Save payment record
	if err := uc.ridesRepo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	// Generate QR code URL for payment processing
	qrCodeURL := fmt.Sprintf("%s?ride_id=%s&amount=%d&passenger_id=%s",
		uc.cfg.Payment.QRCodeBaseURL, req.RideID, adjustedCost, rideObj.PassengerID.String())

	// Create payment request
	paymentRequest := &ride.PaymentRequest{
		RideID:      req.RideID,
		PassengerID: rideObj.PassengerID.String(),
		DriverID:    rideObj.DriverID.String(),
		TotalCost:   adjustedCost,
		QRCodeURL:   qrCodeURL,
	}

	logger.Info("Ride arrived at destination",
		logger.String("ride_id", req.RideID),
		logger.Int("total_cost", adjustedCost),
		logger.String("qr_code_url", qrCodeURL))

	return paymentRequest, nil
}

// ProcessPayment processes the payment for a completed ride
func (uc *rideUC) ProcessPayment(ctx context.Context, req ride.PaymentProccessRequest) (*ride.Payment, error) {
	// Get current ride to verify it exists and is active
	rideObj, err := uc.ridesRepo.GetRide(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	if rideObj.Status != ride.RideStatusOngoing {
		err := fmt.Errorf("cannot process payment for ride that is not ongoing")
		return nil, err
	}

	payment, err := uc.ridesRepo.GetPaymentByRideID(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment record: %w", err)
	}

	// Validate current payment status
	if payment.Status != ride.PaymentStatusPending {
		err := fmt.Errorf("cannot process payment with status: %s", payment.Status)
		return nil, err
	}

	// Validate total cost
	if req.TotalCost != payment.AdjustedCost {
		err := fmt.Errorf("total cost mismatch: expected %d, got %d", payment.AdjustedCost, req.TotalCost)
		return nil, err
	}

	// Update payment status
	payment.Status = req.Status
	err = uc.ridesRepo.UpdatePaymentStatus(ctx, payment.PaymentID.String(), req.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	// Populate driver and passenger IDs for WebSocket notifications
	payment.DriverID = rideObj.DriverID
	payment.PassengerID = rideObj.PassengerID

	// Payment status needs to be accepted for ride to be completed
	if req.Status == ride.PaymentStatusAccepted {
		// Mark ride as completed
		rideObj.Status = ride.RideStatusCompleted
		if err := uc.ridesRepo.CompleteRide(ctx, rideObj); err != nil {
			return nil, fmt.Errorf("failed to mark ride as completed: %w", err)
		}

		// Create ride complete data for the event
		var rideComplete = ride.RideComplete{
			Ride:    *rideObj,
			Payment: *payment,
		}

		// Publish payment processed event
		if err := uc.ridesGW.PublishRideCompleted(ctx, rideComplete); err != nil {
			// Log but don't fail the transaction
			logger.Warn("Failed to publish ride completed event",
				logger.ErrorField(err))
		}
	}

	return payment, nil
}
