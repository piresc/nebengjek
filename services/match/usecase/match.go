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

// ConfirmMatchStatus handles match confirmation from either driver or passenger
// The matchID parameter is currently unused, mp.ID is used instead.
func (uc *MatchUC) ConfirmMatchStatus(matchID string, mp models.MatchProposal) error {
	ctx := context.Background()

	// Scenario 1: Driver accepts, match moves to PENDING_CUSTOMER_CONFIRMATION
	if mp.MatchStatus == models.MatchStatusPendingCustomerConfirmation {
		// Update the match status in Redis to PENDING_CUSTOMER_CONFIRMATION
		// Assuming UpdateMatchStatus updates the status of the match identified by mp.ID
		// and potentially uses DriverID and PassengerID for verification or specific keying.
		if err := uc.matchRepo.UpdateMatchStatus(ctx, mp.ID, models.MatchStatusPendingCustomerConfirmation, mp.DriverID, mp.PassengerID); err != nil {
			return fmt.Errorf("failed to update match status to pending customer confirmation: %w", err)
		}

		// Prepare event for customer notification
		pendingCustomerEvent := mp // mp already has target status
		// mp.MatchStatus is already models.MatchStatusPendingCustomerConfirmation as per the if condition

		// Publish this event using a new gateway method
		if err := uc.matchGW.PublishMatchPendingCustomerConfirmation(ctx, pendingCustomerEvent); err != nil {
			// Log the error but don't necessarily fail the whole operation,
			// as the primary status update in Redis succeeded.
			// Depending on requirements, this could be a hard failure.
			log.Printf("Failed to publish match pending customer confirmation event for MatchID %s: %v", mp.ID, err)
		}

	// Scenario 2: Customer accepts the match (final confirmation)
	} else if mp.MatchStatus == models.MatchStatusAccepted {
		// For acceptance, we need to persist the match to database
		persistedMatch, err := uc.matchRepo.ConfirmAndPersistMatch(ctx, mp.DriverID, mp.PassengerID)
		if err != nil {
			return fmt.Errorf("failed to confirm and persist match: %w", err)
		}

		// Create acceptance event
		acceptEvent := models.MatchProposal{
			ID:             converter.UUIDToStr(persistedMatch.ID),
			PassengerID:    converter.UUIDToStr(persistedMatch.PassengerID),
			DriverID:       converter.UUIDToStr(persistedMatch.DriverID),
			MatchStatus:    models.MatchStatusAccepted, // Final accepted status
			DriverLocation: persistedMatch.DriverLocation,
			UserLocation:   persistedMatch.PassengerLocation,
		}

		// Publish match confirmation
		if err := uc.matchGW.PublishMatchConfirm(ctx, acceptEvent); err != nil {
			return fmt.Errorf("failed to publish match acceptance: %w", err)
		}

		// Remove both users from available pools
		if err := uc.matchRepo.RemoveAvailableDriver(ctx, mp.DriverID); err != nil {
			log.Printf("Failed to remove available driver: %v", err)
		}

		if err := uc.matchRepo.RemoveAvailablePassenger(ctx, mp.PassengerID); err != nil {
			log.Printf("Failed to remove available passenger: %v", err)
		}

	// Scenario 3: Driver or Customer rejects the match
	} else if mp.MatchStatus == models.MatchStatusRejected {
		// For rejections, we simply remove the pending match references
		// These keys will expire automatically, but we clean up for immediate effect
		driverKey := fmt.Sprintf(constants.KeyDriverMatch, mp.DriverID)
		passengerKey := fmt.Sprintf(constants.KeyPassengerMatch, mp.PassengerID)
		pairKey := fmt.Sprintf(constants.KeyPendingMatchPair, mp.DriverID, mp.PassengerID)

		// Delete the keys - we don't care much about errors here as they'll expire anyway
		uc.matchRepo.DeleteRedisKey(ctx, driverKey)
		uc.matchRepo.DeleteRedisKey(ctx, passengerKey)
		uc.matchRepo.DeleteRedisKey(ctx, pairKey)

		// Create rejection event
		rejectEvent := mp // mp already has MatchStatusRejected
		// rejectEvent.MatchStatus = models.MatchStatusRejected // This is already set

		// Publish rejection event
		if err := uc.matchGW.PublishMatchRejected(ctx, rejectEvent); err != nil {
			log.Printf("Failed to publish match rejection: %v", err)
		}
	} else {
		// Handle unknown status
		log.Printf("Unknown match status received: %s for match %s", mp.MatchStatus, mp.ID)
		return fmt.Errorf("unknown match status: %s", mp.MatchStatus)
	}

	return nil
}
