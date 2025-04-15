package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

func (uc *MatchUC) handleActiveDriver(ctx context.Context, event models.BeaconEvent, location *models.Location) error {
	// Add driver to available pool and look for nearby passengers
	if err := uc.matchRepo.AddAvailableDriver(ctx, event.UserID, location); err != nil {
		log.Printf("Failed to add available driver: %v", err)
		return err
	}

	// Find nearby passengers to match with
	nearbyPassengers, err := uc.matchRepo.FindNearbyPassengers(ctx, location, 1.0) // 1km radius
	if err != nil {
		log.Printf("Failed to find nearby passengers: %v", err)
		return err
	}

	// Create match proposals for each nearby passenger
	for _, passenger := range nearbyPassengers {
		match := &models.Match{
			ID:                uuid.New().String(),
			DriverID:          event.UserID,
			PassengerID:       passenger.ID,
			DriverLocation:    event.Location,
			PassengerLocation: passenger.Location,
			Status:            models.MatchStatusPending,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if err := uc.CreateMatch(ctx, match); err != nil {
			log.Printf("Failed to create match with passenger %s: %v", passenger.ID, err)
			continue
		}
	}

	return nil
}

func (uc *MatchUC) handleActivePassenger(ctx context.Context, event models.BeaconEvent, location *models.Location) error {
	if err := uc.matchRepo.AddAvailablePassenger(ctx, event.UserID, location); err != nil {
		log.Printf("Failed to add available passenger: %v", err)
		return err
	}

	// Find nearby drivers to match with
	nearbyDrivers, err := uc.matchRepo.FindNearbyDrivers(ctx, location, 1.0) // 1km radius
	if err != nil {
		log.Printf("Failed to find nearby drivers: %v", err)
		return err
	}

	// Create match proposals for each nearby driver
	for _, driver := range nearbyDrivers {
		match := &models.Match{
			ID:                uuid.New().String(),
			DriverID:          driver.ID,
			PassengerID:       event.UserID,
			DriverLocation:    driver.Location,
			PassengerLocation: event.Location,
			Status:            models.MatchStatusPending,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if err := uc.CreateMatch(ctx, match); err != nil {
			log.Printf("Failed to create match with driver %s: %v", driver.ID, err)
			continue
		}
	}

	return nil
}

func (uc *MatchUC) handleInactiveUser(ctx context.Context, userID string, role string) error {
	var err error
	if role == "driver" {
		err = uc.matchRepo.RemoveAvailableDriver(ctx, userID)
	} else {
		err = uc.matchRepo.RemoveAvailablePassenger(ctx, userID)
	}

	if err != nil {
		log.Printf("Failed to remove available %s: %v", role, err)
		return err
	}
	return nil
}

// HandleBeaconEvent processes beacon events from NATS
func (uc *MatchUC) HandleBeaconEvent(event models.BeaconEvent) error {
	ctx := context.Background()

	if event.IsActive {
		location := &models.Location{
			Latitude:  event.Location.Latitude,
			Longitude: event.Location.Longitude,
		}

		if event.Role == "driver" {
			return uc.handleActiveDriver(ctx, event, location)
		}
		return uc.handleActivePassenger(ctx, event, location)
	}

	return uc.handleInactiveUser(ctx, event.UserID, event.Role)
}

func (uc *MatchUC) CreateMatch(ctx context.Context, match *models.Match) error {
	// Create match in database
	createdMatch, err := uc.matchRepo.CreateMatch(ctx, match)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	// Create match proposal
	matchProposal := models.MatchProposal{
		ID:             createdMatch.ID,
		PassengerID:    createdMatch.PassengerID,
		DriverID:       createdMatch.DriverID,
		UserLocation:   createdMatch.PassengerLocation,
		DriverLocation: createdMatch.DriverLocation,
		MatchStatus:    createdMatch.Status,
	}

	// Publish match proposal event
	if err := uc.matchGW.PublishMatchEvent(ctx, matchProposal); err != nil {
		return fmt.Errorf("failed to publish match proposal: %w", err)
	}

	return nil
}
