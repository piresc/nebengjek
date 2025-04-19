package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
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
func (uc *rideUC) CreateRide(mp models.MatchProposal) error {
	log.Printf("Creating ride for driver %s and customer %s", mp.DriverID, mp.PassengerID)

	// Create a new ride from the match proposal
	ride := &models.Ride{
		DriverID:   uuid.MustParse(mp.DriverID),
		CustomerID: uuid.MustParse(mp.PassengerID),
		Status:     models.RideStatusOngoing,
		TotalCost:  0, // This will be calculated later
	}

	// Delegate to repository
	createdRide, err := uc.ridesRepo.CreateRide(ride)
	if err != nil {
		log.Printf("Failed to create ride: %v", err)
		return err
	}
	err = uc.ridesGW.PublishRideStarted(context.Background(), createdRide)
	if err != nil {
		log.Printf("Failed to publish ride started event: %v", err)
		return err
	}

	log.Printf("Successfully created ride with ID: %s", createdRide.RideID)
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
		log.Printf("Warning: Failed to update total cost for ride %s: %v", rideID, err)
		return fmt.Errorf("failed to update total cost: %w", err)
	}

	log.Printf("Updated billing for ride %s: +%d IDR (%.2f km)", rideID, entry.Cost, entry.Distance)
	return nil
}
