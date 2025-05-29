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

// createMatchesWithNearbyDrivers finds nearby drivers and creates match proposals
func (uc *MatchUC) createMatchesWithNearbyDrivers(ctx context.Context, passengerID string, passengerLocation, targetLocation *models.Location) error {
	nearbyDrivers, err := uc.matchRepo.FindNearbyDrivers(ctx, passengerLocation, uc.cfg.Match.SearchRadiusKm) // Configurable radius
	if err != nil {
		log.Printf("Failed to find nearby drivers: %v", err)
		return err
	}

	// Create match proposals for each nearby driver
	for _, driver := range nearbyDrivers {
		match := uc.buildMatch(driver.ID, passengerID, &driver.Location, passengerLocation, targetLocation)

		if err := uc.CreateMatch(ctx, match); err != nil {
			log.Printf("Failed to create match with driver %s: %v", driver.ID, err)
			continue
		}
	}

	return nil
}

// buildMatch constructs a match object with the provided data
func (uc *MatchUC) buildMatch(driverID, passengerID string, driverLocation, passengerLocation, targetLocation *models.Location) *models.Match {
	match := &models.Match{
		DriverID:          converter.StrToUUID(driverID),
		PassengerID:       converter.StrToUUID(passengerID),
		DriverLocation:    *driverLocation,
		PassengerLocation: *passengerLocation,
		Status:            models.MatchStatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if targetLocation != nil {
		match.TargetLocation = *targetLocation
	}

	return match
}

func (uc *MatchUC) handleActivePassengerWithTarget(ctx context.Context, event models.FinderEvent, location *models.Location, targetLocation *models.Location) error {
	if err := uc.matchRepo.AddAvailablePassenger(ctx, event.UserID, location); err != nil {
		log.Printf("Failed to add available passenger: %v", err)
		return err
	}

	// Find nearby drivers to match with
	return uc.createMatchesWithNearbyDrivers(ctx, event.UserID, location, targetLocation)
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

	location := &models.Location{
		Latitude:  event.Location.Latitude,
		Longitude: event.Location.Longitude,
	}

	if event.IsActive {
		// Beacon events are only for drivers
		return uc.addDriverToPool(ctx, event.UserID, location)
	}

	return uc.handleInactiveUser(ctx, event.UserID, "driver")
}

// HandleFinderEvent processes finder events from NATS for passengers
func (uc *MatchUC) HandleFinderEvent(event models.FinderEvent) error {
	ctx := context.Background()

	location := &models.Location{
		Latitude:  event.Location.Latitude,
		Longitude: event.Location.Longitude,
	}

	targetLocation := &models.Location{
		Latitude:  event.TargetLocation.Latitude,
		Longitude: event.TargetLocation.Longitude,
	}

	if event.IsActive {
		// Finder events are only for passengers who initiate the matching process
		return uc.handleActivePassengerWithTarget(ctx, event, location, targetLocation)
	}

	return uc.handleInactiveUser(ctx, event.UserID, "passenger")
}

// CreateMatch creates a new match and publishes a match proposal event
func (uc *MatchUC) CreateMatch(ctx context.Context, match *models.Match) error {
	// Create match directly in database, which will check for existing pending matches
	createdMatch, err := uc.matchRepo.CreateMatch(ctx, match)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	// Create match proposal for notification
	matchProposal := uc.buildMatchProposal(createdMatch)

	// Publish match proposal event
	if err := uc.matchGW.PublishMatchFound(ctx, matchProposal); err != nil {
		return fmt.Errorf("failed to publish match proposal: %w", err)
	}

	return nil
}

// buildMatchProposal creates a match proposal from a match object
func (uc *MatchUC) buildMatchProposal(match *models.Match) models.MatchProposal {
	return models.MatchProposal{
		ID:             match.ID.String(),
		PassengerID:    converter.UUIDToStr(match.PassengerID),
		DriverID:       converter.UUIDToStr(match.DriverID),
		UserLocation:   match.PassengerLocation,
		DriverLocation: match.DriverLocation,
		TargetLocation: match.TargetLocation,
		MatchStatus:    match.Status,
	}
}

// updateMatchConfirmation updates match confirmation status based on user type
func (uc *MatchUC) updateMatchConfirmation(ctx context.Context, match *models.Match, userID string, isDriver bool) (*models.Match, error) {
	if isDriver {
		match.DriverConfirmed = true
	} else {
		match.PassengerConfirmed = true
	}

	// Determine new status based on confirmations
	if match.DriverConfirmed && match.PassengerConfirmed {
		match.Status = models.MatchStatusAccepted
		log.Printf("Match %s fully confirmed by both parties", match.ID.String())

		// Remove users from available pools when fully confirmed
		uc.matchRepo.RemoveAvailableDriver(ctx, match.DriverID.String())
		uc.matchRepo.RemoveAvailablePassenger(ctx, match.PassengerID.String())
	} else if match.DriverConfirmed {
		match.Status = models.MatchStatusDriverConfirmed
		log.Printf("Match %s confirmed by driver, waiting for passenger", match.ID.String())
	} else if match.PassengerConfirmed {
		match.Status = models.MatchStatusPassengerConfirmed
		log.Printf("Match %s confirmed by passenger, waiting for driver", match.ID.String())
	}

	match.UpdatedAt = time.Now()
	return uc.matchRepo.ConfirmMatchByUser(ctx, match.ID.String(), userID, isDriver)
}

// handleMatchAcceptance processes match acceptance logic
func (uc *MatchUC) handleMatchAcceptance(ctx context.Context, match *models.Match, req *models.MatchConfirmRequest) (models.MatchProposal, error) {
	isDriver := req.UserID == match.DriverID.String()

	updatedMatch, err := uc.updateMatchConfirmation(ctx, match, req.UserID, isDriver)
	if err != nil {
		log.Printf("Warning: Failed to update match confirmation: %v", err)
		updatedMatch = match // Use original match if update fails
	}

	// If match is fully accepted, handle auto-rejection asynchronously
	if updatedMatch.Status == models.MatchStatusAccepted {
		// Start auto-rejection in background to not block HTTP response
		go func() {
			bgCtx := context.Background()
			if err := uc.handleAutoRejectionForAcceptedMatch(bgCtx, updatedMatch); err != nil {
				log.Printf("Warning: Failed to handle auto-rejection: %v", err)
			}
		}()
	}

	responseEvent := uc.buildMatchProposal(updatedMatch)
	log.Printf("Created match proposal response: %+v", responseEvent)

	return responseEvent, nil
}

// handleAutoRejectionForAcceptedMatch rejects all other pending matches for the same passenger
func (uc *MatchUC) handleAutoRejectionForAcceptedMatch(ctx context.Context, acceptedMatch *models.Match) error {
	// Get all pending matches for this passenger
	matches, err := uc.matchRepo.ListMatchesByPassenger(ctx, acceptedMatch.PassengerID)
	if err != nil {
		return fmt.Errorf("failed to list passenger matches: %w", err)
	}

	// Process rejections in batches to reduce database load
	rejectionBatch := make([]string, 0)
	eventBatch := make([]models.MatchProposal, 0)

	for _, otherMatch := range matches {
		// Skip the accepted match
		if otherMatch.ID == acceptedMatch.ID {
			continue
		}

		// Only process if the match is still pending
		if otherMatch.Status == models.MatchStatusPending ||
			otherMatch.Status == models.MatchStatusDriverConfirmed ||
			otherMatch.Status == models.MatchStatusPassengerConfirmed {

			rejectionBatch = append(rejectionBatch, otherMatch.ID.String())

			// Prepare rejection event
			rejectEvent := models.MatchProposal{
				ID:             otherMatch.ID.String(),
				PassengerID:    converter.UUIDToStr(otherMatch.PassengerID),
				DriverID:       converter.UUIDToStr(otherMatch.DriverID),
				MatchStatus:    models.MatchStatusRejected,
				DriverLocation: otherMatch.DriverLocation,
				UserLocation:   otherMatch.PassengerLocation,
				TargetLocation: otherMatch.TargetLocation,
			}
			eventBatch = append(eventBatch, rejectEvent)
		}
	}

	// Batch update match statuses using repository method
	if len(rejectionBatch) > 0 {
		if err := uc.matchRepo.BatchUpdateMatchStatus(ctx, rejectionBatch, models.MatchStatusRejected); err != nil {
			log.Printf("Batch update failed, falling back to individual updates: %v", err)
			// Fallback to individual updates
			for i, matchID := range rejectionBatch {
				if err := uc.matchRepo.UpdateMatchStatus(ctx, matchID, models.MatchStatusRejected); err != nil {
					log.Printf("Failed to update rejected match status for match %s: %v", matchID, err)
					continue
				}

				// Publish rejection event
				if err := uc.matchGW.PublishMatchRejected(ctx, eventBatch[i]); err != nil {
					log.Printf("Failed to publish match rejection for match %s: %v", matchID, err)
				}
			}
		} else {
			// Batch publish events
			for _, event := range eventBatch {
				if err := uc.matchGW.PublishMatchRejected(ctx, event); err != nil {
					log.Printf("Failed to publish match rejection for match %s: %v", event.ID, err)
				}
			}
		}

		log.Printf("Auto-rejected %d matches for passenger %s",
			len(rejectionBatch),
			converter.UUIDToStr(acceptedMatch.PassengerID))
	}

	return nil
}

// handleMatchRejection processes match rejection logic
func (uc *MatchUC) handleMatchRejection(ctx context.Context, match *models.Match) (models.MatchProposal, error) {
	matchID := match.ID.String()

	if err := uc.matchRepo.UpdateMatchStatus(ctx, matchID, models.MatchStatusRejected); err != nil {
		log.Printf("Warning: Failed to update match status to rejected: %v", err)
	}

	// Get updated match to ensure correct state
	updatedMatch, err := uc.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		log.Printf("Warning: Failed to get updated match after rejection: %v", err)
		match.Status = models.MatchStatusRejected // Fallback to local update
		updatedMatch = match
	}

	return uc.buildMatchProposal(updatedMatch), nil
}

// ConfirmMatchStatus handles match confirmation from either driver or passenger
func (uc *MatchUC) ConfirmMatchStatus(req *models.MatchConfirmRequest) (models.MatchProposal, error) {
	ctx := context.Background()

	// Get the match from database
	match, err := uc.matchRepo.GetMatch(ctx, req.ID)
	if err != nil {
		return models.MatchProposal{}, fmt.Errorf("match not found in database: %w", err)
	}

	switch req.Status {
	case string(models.MatchStatusAccepted):
		return uc.handleMatchAcceptance(ctx, match, req)
	case string(models.MatchStatusRejected):
		return uc.handleMatchRejection(ctx, match)
	default:
		return models.MatchProposal{}, fmt.Errorf("unsupported match status: %s", req.Status)
	}
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
