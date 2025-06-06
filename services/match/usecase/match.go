package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/converter"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// addDriverToPool adds a driver to the available pool without creating matches
func (uc *MatchUC) addDriverToPool(ctx context.Context, driverID string, location *models.Location) error {
	// Add driver to available pool
	if err := uc.locationGW.AddAvailableDriver(ctx, driverID, location); err != nil {
		logger.Error("Failed to add available driver",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return err
	}
	return nil
}

// createMatchesWithNearbyDrivers finds nearby drivers and creates match proposals
func (uc *MatchUC) createMatchesWithNearbyDrivers(ctx context.Context, passengerID string, passengerLocation, targetLocation *models.Location) error {
	nearbyDrivers, err := uc.locationGW.FindNearbyDrivers(ctx, passengerLocation, uc.cfg.Match.SearchRadiusKm) // Configurable radius
	if err != nil {
		logger.Error("Failed to find nearby drivers",
			logger.String("passenger_id", passengerID),
			logger.Float64("search_radius_km", uc.cfg.Match.SearchRadiusKm),
			logger.ErrorField(err))
		return err
	}

	// Create match proposals for each nearby driver
	for _, driver := range nearbyDrivers {
		match := uc.buildMatch(driver.ID, passengerID, &driver.Location, passengerLocation, targetLocation)

		if err := uc.CreateMatch(ctx, match); err != nil {
			logger.Error("Failed to create match with driver",
				logger.String("driver_id", driver.ID),
				logger.String("passenger_id", passengerID),
				logger.ErrorField(err))
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
	if err := uc.locationGW.AddAvailablePassenger(ctx, event.UserID, location); err != nil {
		logger.Error("Failed to add available passenger",
			logger.String("passenger_id", event.UserID),
			logger.ErrorField(err))
		return err
	}

	// Find nearby drivers to match with
	return uc.createMatchesWithNearbyDrivers(ctx, event.UserID, location, targetLocation)
}

func (uc *MatchUC) handleInactiveUser(ctx context.Context, userID string, role string) error {
	var err error
	if role == "driver" {
		err = uc.locationGW.RemoveAvailableDriver(ctx, userID)
	} else {
		err = uc.locationGW.RemoveAvailablePassenger(ctx, userID)
	}

	if err != nil {
		logger.Error("Failed to remove available user",
			logger.String("user_id", userID),
			logger.String("role", role),
			logger.ErrorField(err))
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
			logger.Error("Failed to check active ride for driver",
				logger.String("driver_id", event.UserID),
				logger.ErrorField(err))
			// Continue with adding to pool on error to avoid blocking
		} else if hasActiveRide {
			// Driver has active ride, skipping addition to available pool
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
			logger.Error("Failed to check active ride for passenger",
				logger.String("passenger_id", event.UserID),
				logger.ErrorField(err))
			// Continue with adding to pool on error to avoid blocking
		} else if hasActiveRide {
			// Passenger has active ride, skipping addition to available pool
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
		logger.Info("Match fully confirmed by both parties",
			logger.String("match_id", match.ID.String()))

		// Remove users from available pools when fully confirmed
		uc.locationGW.RemoveAvailableDriver(ctx, match.DriverID.String())
		uc.locationGW.RemoveAvailablePassenger(ctx, match.PassengerID.String())
	} else if match.DriverConfirmed {
		match.Status = models.MatchStatusDriverConfirmed
		// Match confirmed by driver, waiting for passenger
	} else if match.PassengerConfirmed {
		match.Status = models.MatchStatusPassengerConfirmed
		// Match confirmed by passenger, waiting for driver
	}

	match.UpdatedAt = time.Now()
	return uc.matchRepo.ConfirmMatchByUser(ctx, match.ID.String(), userID, isDriver)
}

// handleMatchAcceptance processes match acceptance logic
func (uc *MatchUC) handleMatchAcceptance(ctx context.Context, match *models.Match, req *models.MatchConfirmRequest) (models.MatchProposal, error) {
	isDriver := req.UserID == match.DriverID.String()

	updatedMatch, err := uc.updateMatchConfirmation(ctx, match, req.UserID, isDriver)
	if err != nil {
		logger.Warn("Failed to update match confirmation",
			logger.String("match_id", match.ID.String()),
			logger.ErrorField(err))
		updatedMatch = match // Use original match if update fails
	}

	// If match is fully accepted, handle auto-rejection asynchronously
	if updatedMatch.Status == models.MatchStatusAccepted {
		uc.startAsyncAutoRejection(updatedMatch)
		uc.PublishMatchAccepted(updatedMatch)
	}

	responseEvent := uc.buildMatchProposal(updatedMatch)
	// Created match proposal response

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
		logger.Error("Failed to publish match accepted event",
			logger.String("match_id", PublishMatchAccepted.ID),
			logger.ErrorField(err))
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
			logger.Error("Critical: Failed to handle auto-rejection for match",
				logger.String("match_id", match.ID.String()),
				logger.ErrorField(err))
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
		// Auto-rejected matches for passenger
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
		logger.Warn("Batch update failed, falling back to individual updates",
			logger.Int("batch_size", len(rejectionBatch)),
			logger.ErrorField(err))
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
			logger.Error("Failed to update rejected match status",
				logger.String("match_id", matchID),
				logger.ErrorField(err))
			continue
		}

		// Publish rejection event
		if err := uc.matchGW.PublishMatchRejected(ctx, events[i]); err != nil {
			logger.Error("Failed to publish match rejection",
				logger.String("match_id", matchID),
				logger.ErrorField(err))
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
			logger.Error("Failed to publish match rejection",
				logger.String("match_id", event.ID),
				logger.ErrorField(err))
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
		logger.Error("Failed to update match status to rejected",
			logger.String("match_id", matchID),
			logger.ErrorField(err))
	}

	// Get updated match to ensure correct state
	updatedMatch, err := uc.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		logger.Error("Failed to get updated match after rejection",
			logger.String("match_id", matchID),
			logger.ErrorField(err))
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

// ReleaseDriver releases a driver from the ride lock after a ride completes
// The driver will be available for matching again when they send their next beacon event
func (uc *MatchUC) ReleaseDriver(driverID string) error {
	// The locking system only needs to track that the driver is no longer in an active ride
	// The driver will be added back to the available pool when they send their next beacon event
	// with their current location, so we don't need to add them back here

	logger.Info("Successfully released driver from ride lock",
		logger.String("driver_id", driverID))
	return nil
}

// ReleasePassenger releases a passenger from the ride lock after a ride completes
// The passenger will be available for matching again when they send their next finder event
func (uc *MatchUC) ReleasePassenger(passengerID string) error {
	// The locking system only needs to track that the passenger is no longer in an active ride
	// The passenger will be added back to the available pool when they send their next finder event
	// with their current location, so we don't need to add them back here

	logger.Info("Successfully released passenger from ride lock",
		logger.String("passenger_id", passengerID))
	return nil
}

// RemoveDriverFromPool removes a driver from the available pool (locks them)
func (uc *MatchUC) RemoveDriverFromPool(ctx context.Context, driverID string) error {
	// Locking driver (removing from available pool)

	// Remove driver from available pool
	if err := uc.locationGW.RemoveAvailableDriver(ctx, driverID); err != nil {
		logger.Error("Error removing driver from available pool",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to remove driver from available pool: %w", err)
	}

	// Successfully locked driver (removed from available pool)
	return nil
}

// RemovePassengerFromPool removes a passenger from the available pool (locks them)
func (uc *MatchUC) RemovePassengerFromPool(ctx context.Context, passengerID string) error {
	// Locking passenger (removing from available pool)

	// Remove passenger from available pool
	if err := uc.locationGW.RemoveAvailablePassenger(ctx, passengerID); err != nil {
		logger.Error("Error removing passenger from available pool",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to remove passenger from available pool: %w", err)
	}

	// Successfully locked passenger (removed from available pool)
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
