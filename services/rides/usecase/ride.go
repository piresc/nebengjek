package usecase

import (
	"context"
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
		Status:     models.RideStatusPending,
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
