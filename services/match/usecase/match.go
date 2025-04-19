package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

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
	// Create match in database
	createdMatch, err := uc.matchRepo.CreateMatch(ctx, match)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	// Store match proposal in Redis with 5-minute expiration
	if err := uc.matchRepo.StoreMatchProposal(ctx, createdMatch); err != nil {
		return fmt.Errorf("failed to store match proposal: %w", err)
	}

	// Create match proposal
	matchProposal := models.MatchProposal{
		ID:             converter.UUIDToStr(createdMatch.ID),
		PassengerID:    converter.UUIDToStr(createdMatch.PassengerID),
		DriverID:       converter.UUIDToStr(createdMatch.DriverID),
		UserLocation:   createdMatch.PassengerLocation,
		DriverLocation: createdMatch.DriverLocation,
		MatchStatus:    createdMatch.Status,
	}

	// Publish match proposal event
	if err := uc.matchGW.PublishMatchFound(ctx, matchProposal); err != nil {
		return fmt.Errorf("failed to publish match proposal: %w", err)
	}

	return nil
}

// ConfirmMatchStatus handles match confirmation from either driver or passenger
func (uc *MatchUC) ConfirmMatchStatus(matchID string, mp models.MatchProposal) error {
	ctx := context.Background()

	// Get current match to validate user IDs
	match, err := uc.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Validate that the user is part of this match
	if mp.DriverID != converter.UUIDToStr(match.DriverID) && mp.PassengerID != converter.UUIDToStr(match.PassengerID) {
		return fmt.Errorf("user is not part of this match")
	}

	// Use atomic operation to update status
	if err := uc.matchRepo.ConfirmMatchAtomically(ctx, matchID, mp.MatchStatus); err != nil {
		return fmt.Errorf("failed to confirm match: %w", err)
	}

	if mp.MatchStatus == models.MatchStatusAccepted {
		// Publish match confirm event
		acceptEvent := models.MatchProposal{
			ID:             matchID,
			PassengerID:    converter.UUIDToStr(match.PassengerID),
			DriverID:       converter.UUIDToStr(match.DriverID),
			MatchStatus:    models.MatchStatusAccepted,
			DriverLocation: match.DriverLocation,
			UserLocation:   match.PassengerLocation,
		}
		if err := uc.matchGW.PublishMatchConfirm(ctx, acceptEvent); err != nil {
			return fmt.Errorf("failed to publish match acceptance: %w", err)
		}

		// Get all pending matches for this passenger
		matches, err := uc.matchRepo.ListMatchesByPassenger(ctx, match.PassengerID)
		if err != nil {
			return fmt.Errorf("failed to list passenger matches: %w", err)
		}

		// Notify other drivers that their matches were rejected
		for _, otherMatch := range matches {
			// Skip the accepted match
			if converter.UUIDToStr(otherMatch.ID) == matchID {
				continue
			}
			// Only notify if the match is still pending
			if otherMatch.Status == models.MatchStatusPending {
				rejectEvent := models.MatchProposal{
					ID:             converter.UUIDToStr(otherMatch.ID),
					PassengerID:    converter.UUIDToStr(otherMatch.PassengerID),
					DriverID:       converter.UUIDToStr(otherMatch.DriverID),
					MatchStatus:    models.MatchStatusRejected,
					DriverLocation: otherMatch.DriverLocation,
					UserLocation:   otherMatch.PassengerLocation,
				}
				// Update match status to rejected
				if err := uc.matchRepo.UpdateMatchStatus(ctx, converter.UUIDToStr(otherMatch.ID), models.MatchStatusRejected); err != nil {
					log.Printf("Failed to update rejected match status: %v", err)
					continue
				}
				// Publish rejection event
				if err := uc.matchGW.PublishMatchRejected(ctx, rejectEvent); err != nil {
					log.Printf("Failed to publish match rejection: %v", err)
					continue
				}
			}
		}
		err = uc.matchRepo.RemoveAvailableDriver(ctx, mp.DriverID)
		if err != nil {
			log.Printf("Failed to remove available driver: %v", err)
		}
		err = uc.matchRepo.RemoveAvailablePassenger(ctx, mp.PassengerID)
		if err != nil {
			log.Printf("Failed to remove available passenger: %v", err)
		}

	}

	return nil
}
