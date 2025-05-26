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

// addDriverToPool adds a driver to the available pool without creating matches
func (uc *MatchUC) addDriverToPool(ctx context.Context, driverID string, location *models.Location) error {
	// Add driver to available pool
	if err := uc.matchRepo.AddAvailableDriver(ctx, driverID, location); err != nil {
		log.Printf("Failed to add available driver: %v", err)
		return err
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
			// For single-sided matching, drivers only join the pool without creating matches
			return uc.addDriverToPool(ctx, event.UserID, location)
		}
		// Only passengers initiate the matching process
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
func (uc *MatchUC) ConfirmMatchStatus(matchID string, mp models.MatchProposal) (models.MatchProposal, error) {
	ctx := context.Background()

	if mp.MatchStatus == models.MatchStatusAccepted {
		// For acceptance, we need to get the match details first
		match, err := uc.matchRepo.GetPendingMatchByID(ctx, matchID)
		if err != nil {
			return models.MatchProposal{}, fmt.Errorf("failed to get pending match: %w", err)
		}

		// Use the driver and passenger IDs from the match
		driverID := match.DriverID.String()
		passengerID := match.PassengerID.String()

		// Persist the match to database
		persistedMatch, err := uc.matchRepo.ConfirmAndPersistMatch(ctx, driverID, passengerID)
		if err != nil {
			return models.MatchProposal{}, fmt.Errorf("failed to confirm and persist match: %w", err)
		}

		// Create acceptance event
		acceptEvent := models.MatchProposal{
			ID:             converter.UUIDToStr(persistedMatch.ID),
			PassengerID:    converter.UUIDToStr(persistedMatch.PassengerID),
			DriverID:       converter.UUIDToStr(persistedMatch.DriverID),
			MatchStatus:    models.MatchStatusAccepted,
			DriverLocation: persistedMatch.DriverLocation,
			UserLocation:   persistedMatch.PassengerLocation,
		}

		// Remove both users from available pools
		if err := uc.matchRepo.RemoveAvailableDriver(ctx, mp.DriverID); err != nil {
			log.Printf("Failed to remove available driver: %v", err)
		}

		if err := uc.matchRepo.RemoveAvailablePassenger(ctx, mp.PassengerID); err != nil {
			log.Printf("Failed to remove available passenger: %v", err)
		}

		// No longer publishing to NATS - will return directly to HTTP client
		return acceptEvent, nil
	} else {
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
		rejectEvent := mp
		rejectEvent.MatchStatus = models.MatchStatusRejected

		// No longer publishing to NATS - will return directly to HTTP client
		return rejectEvent, nil
	}
}
