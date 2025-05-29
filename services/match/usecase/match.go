package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

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
	// Create match directly in database, which will check for existing pending matches
	createdMatch, err := uc.matchRepo.CreateMatch(ctx, match)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	// Create match proposal for notification
	matchProposal := models.MatchProposal{
		ID:             createdMatch.ID.String(),
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
func (uc *MatchUC) ConfirmMatchStatus(req *models.MatchConfirmRequest) (models.MatchProposal, error) {
	ctx := context.Background()

	// Get the match from database
	match, err := uc.matchRepo.GetMatch(ctx, req.ID)
	if err != nil {
		return models.MatchProposal{}, fmt.Errorf("match not found in database: %w", err)
	}

	driverID := match.DriverID.String()
	passengerID := match.PassengerID.String()
	matchID := match.ID.String()
	isDriver := req.UserID == driverID

	var finalStatus models.MatchStatus = models.MatchStatusPending

	if req.Status == string(models.MatchStatusAccepted) {
		// Update match in database with confirmation
		if isDriver {
			match.DriverConfirmed = true
		} else {
			match.PassengerConfirmed = true
		}

		// Check if both parties have confirmed
		if match.DriverConfirmed && match.PassengerConfirmed {
			finalStatus = models.MatchStatusAccepted
			match.Status = models.MatchStatusAccepted
			log.Printf("Match %s fully confirmed by both parties", matchID)

			// Remove users from available pools
			uc.matchRepo.RemoveAvailableDriver(ctx, driverID)
			uc.matchRepo.RemoveAvailablePassenger(ctx, passengerID)
		} else if match.DriverConfirmed {
			finalStatus = models.MatchStatusDriverConfirmed
			match.Status = models.MatchStatusDriverConfirmed
			log.Printf("Match %s confirmed by driver, waiting for passenger", matchID)
		} else if match.PassengerConfirmed {
			finalStatus = models.MatchStatusPassengerConfirmed
			match.Status = models.MatchStatusPassengerConfirmed
			log.Printf("Match %s confirmed by passenger, waiting for driver", matchID)
		}

		// Update the match in the database
		match.UpdatedAt = time.Now()
		_, err = uc.matchRepo.ConfirmMatchByUser(ctx, matchID, req.UserID, isDriver)
		if err != nil {
			log.Printf("Warning: Failed to update match confirmation: %v", err)
			// Continue anyway to return the response
		}

		// Create response event with updated status
		responseEvent := models.MatchProposal{
			ID:             matchID,
			PassengerID:    passengerID,
			DriverID:       driverID,
			MatchStatus:    finalStatus,
			DriverLocation: match.DriverLocation,
			UserLocation:   match.PassengerLocation,
		}

		return responseEvent, nil
	} else if req.Status == string(models.MatchStatusRejected) {
		// For rejections, update the match status to rejected
		match.Status = models.MatchStatusRejected
		match.UpdatedAt = time.Now()

		err = uc.matchRepo.UpdateMatchStatus(ctx, matchID, models.MatchStatusRejected)
		if err != nil {
			log.Printf("Warning: Failed to update match status to rejected: %v", err)
			// Continue anyway to return the response
		}

		// Create rejection event
		rejectEvent := models.MatchProposal{
			ID:             matchID,
			PassengerID:    passengerID,
			DriverID:       driverID,
			MatchStatus:    models.MatchStatusRejected,
			DriverLocation: match.DriverLocation,
			UserLocation:   match.PassengerLocation,
		}

		return rejectEvent, nil
	}

	return models.MatchProposal{}, fmt.Errorf("unsupported match status: %s", req.Status)
}

// GetMatch retrieves a match by ID
func (uc *MatchUC) GetMatch(ctx context.Context, matchID string) (*models.Match, error) {
	return uc.matchRepo.GetMatch(ctx, matchID)
}

// GetPendingMatch retrieves a pending match by ID
func (uc *MatchUC) GetPendingMatch(ctx context.Context, matchID string) (*models.Match, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to find match: %w", err)
	}

	// Only return if it's in pending state
	if match.Status == models.MatchStatusPending ||
		match.Status == models.MatchStatusDriverConfirmed ||
		match.Status == models.MatchStatusPassengerConfirmed {
		return match, nil
	}

	return nil, fmt.Errorf("match is not in pending state")
}
