package usecase

import (
	"context"
	"fmt"
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
	logger.Info("Creating ride",
		logger.String("driver_id", mp.DriverID),
		logger.String("passenger_id", mp.PassengerID))

	// Create a new ride from the match proposal
	ride := &models.Ride{
		DriverID:    uuid.MustParse(mp.DriverID),
		PassengerID: uuid.MustParse(mp.PassengerID),
		Status:      models.RideStatusDriverPickup, // Set initial status to driver pickup
		TotalCost:   0,                             // This will be calculated later
	}

	// Delegate to repository
	createdRide, err := uc.ridesRepo.CreateRide(ride)
	if err != nil {
		logger.Error("Failed to create ride",
			logger.ErrorField(err))
		return err
	}
	err = uc.ridesGW.PublishRidePickup(context.Background(), createdRide)
	if err != nil {
		logger.Error("Failed to publish ride started event",
			logger.ErrorField(err))
		return err
	}

	logger.Info("Successfully created ride",
		logger.String("ride_id", createdRide.RideID.String()))
	return nil
}

// ProcessBillingUpdate handles billing updates from location aggregates
func (uc *rideUC) ProcessBillingUpdate(rideID string, entry *models.BillingLedger) error {
	ctx := context.Background()

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

// StartTrip updates a ride from driver_pickup to ongoing status
func (uc *rideUC) StartRide(ctx context.Context, req models.RideStartRequest) (*models.Ride, error) {
	// Get current ride to verify it exists and is in pickup state
	ride, err := uc.ridesRepo.GetRide(ctx, req.RideID)
	if err != nil {
		return &models.Ride{}, fmt.Errorf("failed to get ride: %w", err)
	}

	if ride.Status != models.RideStatusDriverPickup {
		return &models.Ride{}, fmt.Errorf("cannot start trip for ride not in driver_pickup state")
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
	// Calculate distance between driver and passenger
	distanceKm := utils.CalculateDistance(driverLoc, passLoc)

	// Convert km to meters
	distanceMeters := distanceKm * 1000

	// Check if driver is close enough to passenger (within 100 meters)
	if distanceMeters > 100 {
		return &models.Ride{}, fmt.Errorf("driver is too far from passenger (%.2f meters)", distanceMeters)
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
		return nil, fmt.Errorf("cannot process arrival for ride that is not ongoing")
	}

	// Get total cost from billing ledger (to ensure accuracy)
	totalCost, err := uc.ridesRepo.GetBillingLedgerSum(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total cost: %w", err)
	}

	// 2. Validate adjustment factor
	if req.AdjustmentFactor < 0 || req.AdjustmentFactor > 1.0 {
		req.AdjustmentFactor = 1.0 // Reset to 100% if invalid
	}

	// 3. Calculate adjusted cost
	adjustedCost := int(float64(totalCost) * req.AdjustmentFactor)

	// 4. Calculate admin fee (5%)
	adminFee := int(float64(adjustedCost) * 0.05)

	// 5. Calculate driver payout
	driverPayout := adjustedCost - adminFee

	// 6. Create payment record
	payment := &models.Payment{
		PaymentID:    uuid.New(),
		RideID:       ride.RideID,
		AdjustedCost: adjustedCost,
		AdminFee:     adminFee,
		DriverPayout: driverPayout,
		Status:       models.PaymentStatusPending, // Set initial status to pending
		CreatedAt:    time.Now(),
	}

	// 7. Save payment record
	if err := uc.ridesRepo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	// Generate QR code URL for payment processing
	// This could be a payment gateway URL with ride and amount parameters
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
		return nil, fmt.Errorf("cannot process payment for ride that is not ongoing")
	}

	payment, err := uc.ridesRepo.GetPaymentByRideID(ctx, req.RideID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment record: %w", err)
	}

	// Validate current payment status
	if payment.Status != models.PaymentStatusPending {
		return nil, fmt.Errorf("cannot process payment with status: %s", payment.Status)
	}

	// validate total cost
	if req.TotalCost != payment.AdjustedCost {
		return nil, fmt.Errorf("total cost mismatch: expected %d, got %d", payment.AdjustedCost, req.TotalCost)
	}

	// update payment status
	payment.Status = req.Status
	err = uc.ridesRepo.UpdatePaymentStatus(ctx, payment.PaymentID.String(), req.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}
	// payment status need to accepted for ride to be completed
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
