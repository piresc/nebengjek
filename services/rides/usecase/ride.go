package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/rides"
)

// RideUC implements the rides.RideUseCase interface
type rideUC struct {
	cfg       *models.Config
	ridesRepo rides.RideRepo
	ridesGW   rides.RideGW
}

// NewRideUC creates a new ride use case
func NewRideUC(
	cfg *models.Config,
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
func (uc *rideUC) CreateRide(ctx context.Context, mp models.MatchProposal) error {
	logger.Info("Creating ride from match proposal",
		logger.String("match_id", mp.ID),
		logger.String("driver_id", mp.DriverID),
		logger.String("passenger_id", mp.PassengerID))

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
	ride := &models.Ride{
		MatchID:     matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.RideStatusDriverPickup, // Set initial status to driver pickup
		TotalCost:   0,                             // This will be calculated later
	}

	logger.Info("Creating ride in database",
		logger.String("match_id", ride.MatchID.String()),
		logger.String("driver_id", ride.DriverID.String()),
		logger.String("passenger_id", ride.PassengerID.String()),
		logger.String("status", string(ride.Status)))

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

	logger.Info("Ride created successfully in database, publishing pickup event",
		logger.String("ride_id", createdRide.RideID.String()),
		logger.String("driver_id", createdRide.DriverID.String()),
		logger.String("passenger_id", createdRide.PassengerID.String()),
		logger.String("status", string(createdRide.Status)))

	err = uc.ridesGW.PublishRidePickup(context.Background(), createdRide)
	if err != nil {
		logger.Error("Failed to publish ride pickup event to NATS",
			logger.String("ride_id", createdRide.RideID.String()),
			logger.String("driver_id", createdRide.DriverID.String()),
			logger.String("passenger_id", createdRide.PassengerID.String()),
			logger.ErrorField(err))
		return err
	}

	logger.Info("Successfully created ride and published pickup event",
		logger.String("ride_id", createdRide.RideID.String()),
		logger.String("driver_id", createdRide.DriverID.String()),
		logger.String("passenger_id", createdRide.PassengerID.String()))
	return nil
}

// ProcessBillingUpdate handles billing updates from location aggregates
func (uc *rideUC) ProcessBillingUpdate(ctx context.Context, rideID string, entry *models.BillingLedger) error {

	// Get current ride to verify it exists and is active
	ride, err := uc.ridesRepo.GetRide(ctx, rideID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	if ride.Status != models.RideStatusOngoing {
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
func (uc *rideUC) StartRide(ctx context.Context, req models.RideStartRequest) (*models.Ride, error) {
	logger.Info("Starting ride request",
		logger.String("ride_id", req.RideID),
		logger.Any("driver_location", req.DriverLocation),
		logger.Any("passenger_location", req.PassengerLocation))

	// Get current ride to verify it exists and is in pickup state
	ride, err := uc.ridesRepo.GetRide(ctx, req.RideID)
	if err != nil {
		logger.Error("Failed to get ride for start request",
			logger.String("ride_id", req.RideID),
			logger.ErrorField(err))
		return &models.Ride{}, fmt.Errorf("failed to get ride: %w", err)
	}

	logger.Info("Retrieved ride for start request",
		logger.String("ride_id", req.RideID),
		logger.String("current_status", string(ride.Status)),
		logger.String("driver_id", ride.DriverID.String()),
		logger.String("passenger_id", ride.PassengerID.String()))

	if ride.Status != models.RideStatusDriverPickup {
		logger.Error("Cannot start ride - invalid status",
			logger.String("ride_id", req.RideID),
			logger.String("current_status", string(ride.Status)),
			logger.String("required_status", string(models.RideStatusDriverPickup)))
		err := fmt.Errorf("cannot start trip for ride not in driver_pickup state, current status: %s", ride.Status)
		return &models.Ride{}, err
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

	// Verify driver is close to passenger (within 100 meters)
	distanceKm := utils.CalculateDistance(driverLoc, passLoc)
	distanceMeters := distanceKm * 1000

	logger.Info("Calculated distance between driver and passenger",
		logger.String("ride_id", req.RideID),
		logger.Float64("distance_meters", distanceMeters),
		logger.Float64("max_allowed_meters", 100))

	// Check if driver is close enough to passenger (within 100 meters)
	if distanceMeters > 100 {
		logger.Error("Driver too far from passenger",
			logger.String("ride_id", req.RideID),
			logger.Float64("distance_meters", distanceMeters),
			logger.Float64("max_allowed_meters", 100),
			logger.Any("driver_location", req.DriverLocation),
			logger.Any("passenger_location", req.PassengerLocation))
		err := fmt.Errorf("driver is too far from passenger (%.2f meters)", distanceMeters)
		return &models.Ride{}, err
	}

	// Update ride status to ongoing
	ride.Status = models.RideStatusOngoing
	if err := uc.ridesRepo.UpdateRideStatus(ctx, ride.RideID.String(), models.RideStatusOngoing); err != nil {
		return &models.Ride{}, fmt.Errorf("failed to update ride status to ongoing: %w", err)
	}

	logger.Info("Ride started - Driver picked up passenger",
		logger.String("ride_id", req.RideID))
	return ride, nil
}

// RideArrived handles when a ride arrives at the destination but before payment processing
func (uc *rideUC) RideArrived(ctx context.Context, req models.RideArrivalReq) (*models.PaymentRequest, error) {
	// Get current ride to verify it exists and is active
	ride, err := uc.ridesRepo.GetRide(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	if ride.Status != models.RideStatusOngoing {
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
	payment := &models.Payment{
		PaymentID:    uuid.New(),
		RideID:       ride.RideID,
		AdjustedCost: adjustedCost,
		AdminFee:     adminFee,
		DriverPayout: driverPayout,
		Status:       models.PaymentStatusPending,
		CreatedAt:    time.Now(),
	}

	// Save payment record
	if err := uc.ridesRepo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	// Generate QR code URL for payment processing
	qrCodeURL := fmt.Sprintf("%s?ride_id=%s&amount=%d&passenger_id=%s",
		uc.cfg.Payment.QRCodeBaseURL, req.RideID, adjustedCost, ride.PassengerID.String())

	// Create payment request
	paymentRequest := &models.PaymentRequest{
		RideID:      req.RideID,
		PassengerID: ride.PassengerID.String(),
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
func (uc *rideUC) ProcessPayment(ctx context.Context, req models.PaymentProccessRequest) (*models.Payment, error) {
	// Get current ride to verify it exists and is active
	ride, err := uc.ridesRepo.GetRide(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	if ride.Status != models.RideStatusOngoing {
		err := fmt.Errorf("cannot process payment for ride that is not ongoing")
		return nil, err
	}

	payment, err := uc.ridesRepo.GetPaymentByRideID(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment record: %w", err)
	}

	// Validate current payment status
	if payment.Status != models.PaymentStatusPending {
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

	// Payment status needs to be accepted for ride to be completed
	if req.Status == models.PaymentStatusAccepted {
		// Mark ride as completed
		ride.Status = models.RideStatusCompleted
		if err := uc.ridesRepo.CompleteRide(ctx, ride); err != nil {
			return nil, fmt.Errorf("failed to mark ride as completed: %w", err)
		}

		// Create ride complete data for the event
		var rideComplete = models.RideComplete{
			Ride:    *ride,
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
