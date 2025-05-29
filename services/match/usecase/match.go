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
	nearbyDrivers, err := uc.matchRepo.FindNearbyDrivers(ctx, location, uc.cfg.Match.SearchRadiusKm) // Configurable radius
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

func (uc *MatchUC) handleActivePassengerWithTarget(ctx context.Context, event models.FinderEvent, location *models.Location, targetLocation *models.Location) error {
	if err := uc.matchRepo.AddAvailablePassenger(ctx, event.UserID, location); err != nil {
		log.Printf("Failed to add available passenger: %v", err)
		return err
	}

	// Find nearby drivers to match with
	nearbyDrivers, err := uc.matchRepo.FindNearbyDrivers(ctx, location, uc.cfg.Match.SearchRadiusKm) // Configurable radius
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
			PassengerLocation: *location,
			TargetLocation:    *targetLocation,
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

// HandleBeaconEvent processes beacon events from NATS for drivers
func (uc *MatchUC) HandleBeaconEvent(event models.BeaconEvent) error {
	ctx := context.Background()

	if event.IsActive {
		location := &models.Location{
			Latitude:  event.Location.Latitude,
			Longitude: event.Location.Longitude,
		}

		// Beacon events are only for drivers
		return uc.addDriverToPool(ctx, event.UserID, location)
	}

	return uc.handleInactiveUser(ctx, event.UserID, "driver")
}

// HandleFinderEvent processes finder events from NATS for passengers
func (uc *MatchUC) HandleFinderEvent(event models.FinderEvent) error {
	ctx := context.Background()

	if event.IsActive {
		location := &models.Location{
			Latitude:  event.Location.Latitude,
			Longitude: event.Location.Longitude,
		}

		targetLocation := &models.Location{
			Latitude:  event.TargetLocation.Latitude,
			Longitude: event.TargetLocation.Longitude,
		}

		// Finder events are only for passengers who initiate the matching process
		return uc.handleActivePassengerWithTarget(ctx, event, location, targetLocation)
	}

	return uc.handleInactiveUser(ctx, event.UserID, "passenger")
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
		TargetLocation: createdMatch.TargetLocation,
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

	if req.Status == string(models.MatchStatusAccepted) {
		// Update match in database with confirmation
		if isDriver {
			match.DriverConfirmed = true
		} else {
			match.PassengerConfirmed = true
		}

		// Check if both parties have confirmed
		if match.DriverConfirmed && match.PassengerConfirmed {
			match.Status = models.MatchStatusAccepted
			log.Printf("Match %s fully confirmed by both parties", matchID)

			// Remove users from available pools
			uc.matchRepo.RemoveAvailableDriver(ctx, driverID)
			uc.matchRepo.RemoveAvailablePassenger(ctx, passengerID)
		} else if match.DriverConfirmed {
			match.Status = models.MatchStatusDriverConfirmed
			log.Printf("Match %s confirmed by driver, waiting for passenger", matchID)
		} else if match.PassengerConfirmed {
			match.Status = models.MatchStatusPassengerConfirmed
			log.Printf("Match %s confirmed by passenger, waiting for driver", matchID)
		}

		// Update the match in the database
		match.UpdatedAt = time.Now()
		updatedMatch, err := uc.matchRepo.ConfirmMatchByUser(ctx, matchID, req.UserID, isDriver)
		if err != nil {
			log.Printf("Warning: Failed to update match confirmation: %v", err)
			// Continue anyway to return the response
		} else {
			// Use the updated match from the repository to ensure status is correct
			match = updatedMatch
		}

		// Create response event with updated status from the match
		responseEvent := models.MatchProposal{
			ID:             matchID,
			PassengerID:    passengerID,
			DriverID:       driverID,
			MatchStatus:    match.Status,
			DriverLocation: match.DriverLocation,
			UserLocation:   match.PassengerLocation,
			TargetLocation: match.TargetLocation,
		}

		// Log the response event to help with debugging
		log.Printf("Created match proposal response: %+v", responseEvent)
		log.Printf("Driver location: %+v", match.DriverLocation)
		log.Printf("Passenger location: %+v", match.PassengerLocation)
		log.Printf("Target location: %+v", match.TargetLocation)

		return responseEvent, nil
	} else if req.Status == string(models.MatchStatusRejected) {
		// For rejections, update the match status to rejected
		match.Status = models.MatchStatusRejected
		match.UpdatedAt = time.Now()

		err := uc.matchRepo.UpdateMatchStatus(ctx, matchID, models.MatchStatusRejected)
		if err != nil {
			log.Printf("Warning: Failed to update match status to rejected: %v", err)
			// Continue anyway to return the response
		}

		// After updating status, get the latest match to ensure we have the correct state
		updatedMatch, err := uc.matchRepo.GetMatch(ctx, matchID)
		if err != nil {
			log.Printf("Warning: Failed to get updated match after rejection: %v", err)
			// Continue with local match object if we couldn't fetch the updated one
		} else {
			match = updatedMatch
		}

		// Create rejection event with the updated match status
		rejectEvent := models.MatchProposal{
			ID:             matchID,
			PassengerID:    passengerID,
			DriverID:       driverID,
			MatchStatus:    match.Status,
			DriverLocation: match.DriverLocation,
			UserLocation:   match.PassengerLocation,
			TargetLocation: match.TargetLocation,
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
