package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/converter"
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
			DriverID:          converter.StrToUUID(event.UserID),
			PassengerID:       converter.StrToUUID(passenger.ID),
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
			DriverID:          converter.StrToUUID(driver.ID),
			PassengerID:       converter.StrToUUID(event.UserID),
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
	// Create pending match in Redis with 1-minute expiration using SetNX
	// This ensures we don't create duplicate matches between the same driver and passenger
	matchID, err := uc.matchRepo.CreatePendingMatch(ctx, match)
	if err != nil {
		return fmt.Errorf("failed to create pending match: %w", err)
	}

	// If matchID is empty, it means a match already exists between this driver and passenger
	if matchID == "" {
		// Skip creating another match proposal
		log.Printf("Match already exists between driver %s and passenger %s",
			converter.UUIDToStr(match.DriverID), converter.UUIDToStr(match.PassengerID))
		return nil
	}

	// Create match proposal for notification
	matchProposal := models.MatchProposal{
		ID:             matchID,
		PassengerID:    converter.UUIDToStr(match.PassengerID),
		DriverID:       converter.UUIDToStr(match.DriverID),
		UserLocation:   match.PassengerLocation,
		DriverLocation: match.DriverLocation,
		MatchStatus:    models.MatchStatusPending,
	}

	// Publish match proposal event
	if err := uc.matchGW.PublishMatchFound(ctx, matchProposal); err != nil {
		return fmt.Errorf("failed to publish match proposal: %w", err)
	}

	return nil
}

// ConfirmMatchStatus handles different stages of match confirmation based on mp.MatchStatus.
func (uc *MatchUC) ConfirmMatchStatus(mp models.MatchProposal) error {
	ctx := context.Background()

	switch mp.MatchStatus {
	case models.MatchStatusPendingCustomerConfirmation:
		// Driver has accepted, waiting for customer's final confirmation.
		// Update status in repository (e.g., Redis).
		if err := uc.matchRepo.UpdateMatchStatus(ctx, mp.ID, models.MatchStatusPendingCustomerConfirmation, mp.DriverID, mp.PassengerID); err != nil {
			return fmt.Errorf("failed to update match status to pending customer confirmation: %w", err)
		}

		// Publish event to notify relevant services (e.g., User service to inform passenger).
		if err := uc.matchGW.PublishMatchPendingCustomerConfirmation(ctx, mp); err != nil {
			log.Printf("Error publishing MatchPendingCustomerConfirmation for MatchID %s: %v. Propagating error.", mp.ID, err)
			return err
		}

	case models.MatchStatusAccepted:
		// Customer has accepted the match (final confirmation).
		// Persist the match to the database.
		persistedMatch, err := uc.matchRepo.ConfirmAndPersistMatch(ctx, mp.DriverID, mp.PassengerID)
		if err != nil {
			return fmt.Errorf("failed to confirm and persist match: %w", err)
		}

		// Prepare event for NATS publication, using details from the persisted match.
		acceptEvent := models.MatchProposal{
			ID:             converter.UUIDToStr(persistedMatch.ID),
			PassengerID:    converter.UUIDToStr(persistedMatch.PassengerID),
			DriverID:       converter.UUIDToStr(persistedMatch.DriverID),
			MatchStatus:    models.MatchStatusAccepted,
			DriverLocation: persistedMatch.DriverLocation,
			UserLocation:   persistedMatch.PassengerLocation,
		}

		if err := uc.matchGW.PublishMatchConfirm(ctx, acceptEvent); err != nil {
			// Error already includes "failed to publish match acceptance" or similar from gateway.
			return err
		}

		// Remove both users from their respective available pools.
		// Errors here are logged but not returned, as the match confirmation itself was successful.
		if err := uc.matchRepo.RemoveAvailableDriver(ctx, mp.DriverID); err != nil {
			log.Printf("Failed to remove available driver %s after match confirmation: %v", mp.DriverID, err)
		}
		if err := uc.matchRepo.RemoveAvailablePassenger(ctx, mp.PassengerID); err != nil {
			log.Printf("Failed to remove available passenger %s after match confirmation: %v", mp.PassengerID, err)
		}

	case models.MatchStatusRejected:
		// Driver or Customer has rejected the match.
		// Remove pending match references from Redis.
		// These are best-effort deletions; keys will expire naturally if deletion fails.
		// Thus, errors from DeleteRedisKey are logged but not propagated.
		driverKey := fmt.Sprintf(constants.KeyDriverMatch, mp.DriverID)
		passengerKey := fmt.Sprintf(constants.KeyPassengerMatch, mp.PassengerID)
		pairKey := fmt.Sprintf(constants.KeyPendingMatchPair, mp.DriverID, mp.PassengerID)

		if err := uc.matchRepo.DeleteRedisKey(ctx, driverKey); err != nil {
			log.Printf("Error deleting Redis key %s for rejected match %s: %v", driverKey, mp.ID, err)
		}
		if err := uc.matchRepo.DeleteRedisKey(ctx, passengerKey); err != nil {
			log.Printf("Error deleting Redis key %s for rejected match %s: %v", passengerKey, mp.ID, err)
		}
		if err := uc.matchRepo.DeleteRedisKey(ctx, pairKey); err != nil {
			log.Printf("Error deleting Redis key %s for rejected match %s: %v", pairKey, mp.ID, err)
		}
		
		// Publish rejection event.
		if err := uc.matchGW.PublishMatchRejected(ctx, mp); err != nil {
			log.Printf("Error publishing MatchRejected for MatchID %s: %v. Propagating error.", mp.ID, err)
			return err
		}

	default:
		// Handle unknown or unsupported match status.
		log.Printf("Unknown match status received: %s for match %s", mp.MatchStatus, mp.ID)
		return fmt.Errorf("unknown match status: %s", mp.MatchStatus)
	}

	return nil
}
