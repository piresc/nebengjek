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
		// Check if driver has an active ride before adding to pool
		hasActiveRide, err := uc.HasActiveRide(ctx, event.UserID, true) // true = isDriver
		if err != nil {
			log.Printf("Failed to check active ride for driver %s: %v", event.UserID, err)
			// Continue with adding to pool on error to avoid blocking
		} else if hasActiveRide {
			log.Printf("Driver %s has active ride, skipping addition to available pool", event.UserID)
			return nil
		}

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
		// Check if passenger has an active ride before adding to pool
		hasActiveRide, err := uc.HasActiveRide(ctx, event.UserID, false) // false = isPassenger
		if err != nil {
			log.Printf("Failed to check active ride for passenger %s: %v", event.UserID, err)
			// Continue with adding to pool on error to avoid blocking
		} else if hasActiveRide {
			log.Printf("Passenger %s has active ride, skipping addition to available pool", event.UserID)
			return nil
		}

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
		uc.startAsyncAutoRejection(updatedMatch)
		uc.PublishMatchAccepted(updatedMatch)
	}

	responseEvent := uc.buildMatchProposal(updatedMatch)
	log.Printf("Created match proposal response: %+v", responseEvent)

	return responseEvent, nil
}

func (uc *MatchUC) PublishMatchAccepted(match *models.Match) {
	// Create match proposal for accepted match
	PublishMatchAccepted := models.MatchProposal{
		ID:             match.ID.String(),
		PassengerID:    converter.UUIDToStr(match.PassengerID),
		DriverID:       converter.UUIDToStr(match.DriverID),
		UserLocation:   match.PassengerLocation,
		DriverLocation: match.DriverLocation,
		TargetLocation: match.TargetLocation,
		MatchStatus:    match.Status,
	}
	if err := uc.matchGW.PublishMatchAccepted(context.Background(), PublishMatchAccepted); err != nil {
		log.Printf("Failed to publish match accepted event: %v", err)
	}
}

// startAsyncAutoRejection initiates the asynchronous auto-rejection process
func (uc *MatchUC) startAsyncAutoRejection(match *models.Match) {
	// Create context with timeout for background operation
	bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Start auto-rejection in background with proper context management
	go func() {
		defer cancel() // Ensure context is cleaned up

		if err := uc.handleAutoRejectionForAcceptedMatch(bgCtx, match); err != nil {
			log.Printf("Critical: Failed to handle auto-rejection for match %s: %v",
				match.ID.String(), err)
			// TODO: Add alerting/retry mechanism for critical failures
		}
	}()
}

// handleAutoRejectionForAcceptedMatch rejects all other pending matches for the same passenger
func (uc *MatchUC) handleAutoRejectionForAcceptedMatch(ctx context.Context, acceptedMatch *models.Match) error {
	// Add timeout check
	select {
	case <-ctx.Done():
		return fmt.Errorf("auto-rejection cancelled: %w", ctx.Err())
	default:
	}

	// Get all pending matches for this passenger
	matches, err := uc.matchRepo.ListMatchesByPassenger(ctx, acceptedMatch.PassengerID)
	if err != nil {
		return fmt.Errorf("failed to list passenger matches: %w", err)
	}

	// Process rejections in batches to reduce database load
	rejectionBatch := make([]string, 0)
	eventBatch := make([]models.MatchProposal, 0)

	for _, otherMatch := range matches {
		// Check context again during processing
		select {
		case <-ctx.Done():
			return fmt.Errorf("auto-rejection cancelled during processing: %w", ctx.Err())
		default:
		}

		// Skip the accepted match
		if otherMatch.ID == acceptedMatch.ID {
			continue
		}

		// Only process if the match is still pending
		if otherMatch.Status == models.MatchStatusPending ||
			otherMatch.Status == models.MatchStatusDriverConfirmed ||
			otherMatch.Status == models.MatchStatusPassengerConfirmed {

			rejectionBatch = append(rejectionBatch, otherMatch.ID.String())
			eventBatch = append(eventBatch, uc.createRejectionEvent(otherMatch))
		}
	}

	// Process rejection batch and publish events
	if err := uc.processRejectionBatch(ctx, rejectionBatch, eventBatch); err != nil {
		return fmt.Errorf("failed to process rejection batch: %w", err)
	}

	if len(rejectionBatch) > 0 {
		log.Printf("Auto-rejected %d matches for passenger %s",
			len(rejectionBatch),
			converter.UUIDToStr(acceptedMatch.PassengerID))
	}

	return nil
}

// processRejectionBatch handles the batch update of rejected matches and event publishing
func (uc *MatchUC) processRejectionBatch(ctx context.Context, rejectionBatch []string, eventBatch []models.MatchProposal) error {
	if len(rejectionBatch) == 0 {
		return nil
	}

	// Attempt batch update
	if err := uc.matchRepo.BatchUpdateMatchStatus(ctx, rejectionBatch, models.MatchStatusRejected); err != nil {
		log.Printf("Batch update failed, falling back to individual updates: %v", err)
		return uc.processIndividualRejections(ctx, rejectionBatch, eventBatch)
	}

	// Batch publish events
	return uc.publishRejectionEvents(ctx, eventBatch)
}

// processIndividualRejections handles individual updates when batch update fails
func (uc *MatchUC) processIndividualRejections(ctx context.Context, matchIDs []string, events []models.MatchProposal) error {
	for i, matchID := range matchIDs {
		select {
		case <-ctx.Done():
			return fmt.Errorf("auto-rejection cancelled during fallback updates: %w", ctx.Err())
		default:
		}

		if err := uc.matchRepo.UpdateMatchStatus(ctx, matchID, models.MatchStatusRejected); err != nil {
			log.Printf("Failed to update rejected match status for match %s: %v", matchID, err)
			continue
		}

		// Publish rejection event
		if err := uc.matchGW.PublishMatchRejected(ctx, events[i]); err != nil {
			log.Printf("Failed to publish match rejection for match %s: %v", matchID, err)
		}
	}
	return nil
}

// publishRejectionEvents publishes all rejection events
func (uc *MatchUC) publishRejectionEvents(ctx context.Context, events []models.MatchProposal) error {
	for _, event := range events {
		select {
		case <-ctx.Done():
			return fmt.Errorf("auto-rejection cancelled during event publishing: %w", ctx.Err())
		default:
		}

		if err := uc.matchGW.PublishMatchRejected(ctx, event); err != nil {
			log.Printf("Failed to publish match rejection for match %s: %v", event.ID, err)
		}
	}
	return nil
}

// createRejectionEvent creates a match proposal event for rejection
func (uc *MatchUC) createRejectionEvent(match *models.Match) models.MatchProposal {
	return models.MatchProposal{
		ID:             match.ID.String(),
		PassengerID:    converter.UUIDToStr(match.PassengerID),
		DriverID:       converter.UUIDToStr(match.DriverID),
		MatchStatus:    models.MatchStatusRejected,
		DriverLocation: match.DriverLocation,
		UserLocation:   match.PassengerLocation,
		TargetLocation: match.TargetLocation,
	}
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

// ReleaseDriver adds a driver back to the available pool after a ride completes
func (uc *MatchUC) ReleaseDriver(driverID string) error {
	ctx := context.Background()

	// Add driver back to the available pool
	log.Printf("Releasing driver %s back to available pool", driverID)

	// Get driver's last known location (if any)
	location, err := uc.matchRepo.GetDriverLocation(ctx, driverID)
	if err != nil {
		log.Printf("Warning: Failed to get driver location: %v", err)
		// If we can't get the location, we can't add them back to the geo index
		return fmt.Errorf("failed to get driver location: %w", err)
	}

	// Re-add driver to available pool with their last known location
	if err := uc.matchRepo.AddAvailableDriver(ctx, driverID, &location); err != nil {
		log.Printf("Error adding driver back to available pool: %v", err)
		return fmt.Errorf("failed to add driver back to available pool: %w", err)
	}

	log.Printf("Successfully released driver %s back to available pool", driverID)
	return nil
}

// ReleasePassenger adds a passenger back to the available pool after a ride completes
func (uc *MatchUC) ReleasePassenger(passengerID string) error {
	ctx := context.Background()

	// Add passenger back to the available pool
	log.Printf("Releasing passenger %s back to available pool", passengerID)

	// Get passenger's last known location (if any)
	location, err := uc.matchRepo.GetPassengerLocation(ctx, passengerID)
	if err != nil {
		log.Printf("Warning: Failed to get passenger location: %v", err)
		// If we can't get the location, we can't add them back to the geo index
		return fmt.Errorf("failed to get passenger location: %w", err)
	}

	// Re-add passenger to available pool with their last known location
	if err := uc.matchRepo.AddAvailablePassenger(ctx, passengerID, &location); err != nil {
		log.Printf("Error adding passenger back to available pool: %v", err)
		return fmt.Errorf("failed to add passenger back to available pool: %w", err)
	}

	log.Printf("Successfully released passenger %s back to available pool", passengerID)
	return nil
}

// RemoveDriverFromPool removes a driver from the available pool (locks them)
func (uc *MatchUC) RemoveDriverFromPool(ctx context.Context, driverID string) error {
	log.Printf("Locking driver %s (removing from available pool)", driverID)

	// Remove driver from available pool
	if err := uc.matchRepo.RemoveAvailableDriver(ctx, driverID); err != nil {
		log.Printf("Error removing driver from available pool: %v", err)
		return fmt.Errorf("failed to remove driver from available pool: %w", err)
	}

	log.Printf("Successfully locked driver %s (removed from available pool)", driverID)
	return nil
}

// RemovePassengerFromPool removes a passenger from the available pool (locks them)
func (uc *MatchUC) RemovePassengerFromPool(ctx context.Context, passengerID string) error {
	log.Printf("Locking passenger %s (removing from available pool)", passengerID)

	// Remove passenger from available pool
	if err := uc.matchRepo.RemoveAvailablePassenger(ctx, passengerID); err != nil {
		log.Printf("Error removing passenger from available pool: %v", err)
		return fmt.Errorf("failed to remove passenger from available pool: %w", err)
	}

	log.Printf("Successfully locked passenger %s (removed from available pool)", passengerID)
	return nil
}

// SetActiveRide stores active ride information for both driver and passenger
func (uc *MatchUC) SetActiveRide(ctx context.Context, driverID, passengerID, rideID string) error {
	return uc.matchRepo.SetActiveRide(ctx, driverID, passengerID, rideID)
}

// RemoveActiveRide removes active ride information for both driver and passenger
func (uc *MatchUC) RemoveActiveRide(ctx context.Context, driverID, passengerID string) error {
	return uc.matchRepo.RemoveActiveRide(ctx, driverID, passengerID)
}

// HasActiveRide checks if a user (driver or passenger) has an active ride
func (uc *MatchUC) HasActiveRide(ctx context.Context, userID string, isDriver bool) (bool, error) {
	var rideID string
	var err error

	if isDriver {
		rideID, err = uc.matchRepo.GetActiveRideByDriver(ctx, userID)
	} else {
		rideID, err = uc.matchRepo.GetActiveRideByPassenger(ctx, userID)
	}

	if err != nil {
		return false, fmt.Errorf("failed to check active ride: %w", err)
	}

	// If rideID is empty, no active ride exists
	return rideID != "", nil
}
