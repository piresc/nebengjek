package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// addDriverToPool adds a driver to the available pool without creating matches
func (uc *MatchUC) addDriverToPool(ctx context.Context, driverID string, loc *locationmodels.Location) error {
	// Add driver to available pool
	if err := uc.matchGW.AddAvailableDriver(ctx, driverID, loc); err != nil {
		logger.Error("Failed to add available driver",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		return err
	}
	return nil
}

// createMatchesWithNearbyDrivers finds nearby drivers and creates match proposals
func (uc *MatchUC) createMatchesWithNearbyDrivers(ctx context.Context, passengerID string, passengerLocation, targetLocation *locationmodels.Location) error {
	nearbyDrivers, err := uc.matchGW.FindNearbyDrivers(ctx, passengerLocation, uc.cfg.Match.SearchRadiusKm) // Configurable radius
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
func (uc *MatchUC) buildMatch(driverID, passengerID string, driverLoc, passengerLoc, targetLoc *locationmodels.Location) *matchmodels.Match {
	logger.Info("Building match with locations",
		logger.String("driver_id", driverID),
		logger.String("passenger_id", passengerID),
		logger.Float64("driver_lat", driverLoc.Latitude),
		logger.Float64("driver_lon", driverLoc.Longitude),
		logger.Float64("passenger_lat", passengerLoc.Latitude),
		logger.Float64("passenger_lon", passengerLoc.Longitude))

	if targetLoc != nil {
		logger.Info("Target location provided",
			logger.Float64("target_lat", targetLoc.Latitude),
			logger.Float64("target_lon", targetLoc.Longitude))
	} else {
		logger.Warn("Target location is nil")
	}

	match := &matchmodels.Match{
		DriverID:          core.StrToUUID(driverID),
		PassengerID:       core.StrToUUID(passengerID),
		DriverLocation:    *driverLoc,
		PassengerLocation: *passengerLoc,
		Status:            matchmodels.MatchStatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if targetLoc != nil {
		match.TargetLocation = *targetLoc
		logger.Info("Match created with target location",
			logger.Float64("match_target_lat", match.TargetLocation.Latitude),
			logger.Float64("match_target_lon", match.TargetLocation.Longitude))
	}

	return match
}

func (uc *MatchUC) handleActivePassengerWithTarget(ctx context.Context, event core.FinderEvent, loc *locationmodels.Location, targetLoc *locationmodels.Location) error {
	if err := uc.matchGW.AddAvailablePassenger(ctx, event.UserID, loc); err != nil {
		logger.Error("Failed to add available passenger",
			logger.String("passenger_id", event.UserID),
			logger.ErrorField(err))
		return err
	}

	// Find nearby drivers to match with
	return uc.createMatchesWithNearbyDrivers(ctx, event.UserID, loc, targetLoc)
}

func (uc *MatchUC) handleInactiveUser(ctx context.Context, userID string, role string) error {
	var err error
	if role == "driver" {
		err = uc.matchGW.RemoveAvailableDriver(ctx, userID)
	} else {
		err = uc.matchGW.RemoveAvailablePassenger(ctx, userID)
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
func (uc *MatchUC) HandleBeaconEvent(ctx context.Context, event core.BeaconEvent) error {

	location := &locationmodels.Location{
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
func (uc *MatchUC) HandleFinderEvent(ctx context.Context, event core.FinderEvent) error {

	location := &locationmodels.Location{
		Latitude:  event.Location.Latitude,
		Longitude: event.Location.Longitude,
	}

	targetLocation := &locationmodels.Location{
		Latitude:  event.TargetLocation.Latitude,
		Longitude: event.TargetLocation.Longitude,
	}

	logger.InfoCtx(ctx, "Processing finder event with target location",
		logger.String("user_id", event.UserID),
		logger.Bool("is_active", event.IsActive),
		logger.Float64("event_target_lat", event.TargetLocation.Latitude),
		logger.Float64("event_target_lon", event.TargetLocation.Longitude),
		logger.Float64("processed_target_lat", targetLocation.Latitude),
		logger.Float64("processed_target_lon", targetLocation.Longitude))

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
func (uc *MatchUC) CreateMatch(ctx context.Context, match *matchmodels.Match) error {
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
func (uc *MatchUC) buildMatchProposal(match *matchmodels.Match) matchmodels.MatchProposal {
	logger.Info("Building match proposal",
		logger.String("match_id", match.ID.String()),
		logger.Float64("match_target_lat", match.TargetLocation.Latitude),
		logger.Float64("match_target_lon", match.TargetLocation.Longitude),
		logger.String("match_target_location", fmt.Sprintf("%.6f,%.6f", match.TargetLocation.Latitude, match.TargetLocation.Longitude)))

	proposal := matchmodels.MatchProposal{
		ID:             match.ID.String(),
		PassengerID:    core.UUIDToStr(match.PassengerID),
		DriverID:       core.UUIDToStr(match.DriverID),
		UserLocation:   match.PassengerLocation,
		DriverLocation: match.DriverLocation,
		TargetLocation: match.TargetLocation,
		MatchStatus:    match.Status,
	}

	logger.Info("Match proposal created",
		logger.Float64("proposal_target_lat", proposal.TargetLocation.Latitude),
		logger.Float64("proposal_target_lon", proposal.TargetLocation.Longitude),
		logger.String("proposal_target_location", fmt.Sprintf("%.6f,%.6f", proposal.TargetLocation.Latitude, proposal.TargetLocation.Longitude)))

	return proposal
}

// updateMatchConfirmation updates match confirmation status based on user type
func (uc *MatchUC) updateMatchConfirmation(ctx context.Context, match *matchmodels.Match, userID string, isDriver bool) (*matchmodels.Match, error) {
	if isDriver {
		match.DriverConfirmed = true
	} else {
		match.PassengerConfirmed = true
	}

	// Determine new status based on confirmations
	if match.DriverConfirmed && match.PassengerConfirmed {
		match.Status = matchmodels.MatchStatusAccepted
		logger.Info("Match fully confirmed by both parties",
			logger.String("match_id", match.ID.String()))

		// Remove users from available pools when fully confirmed
		uc.matchGW.RemoveAvailableDriver(ctx, match.DriverID.String())
		uc.matchGW.RemoveAvailablePassenger(ctx, match.PassengerID.String())
	} else if match.DriverConfirmed {
		match.Status = matchmodels.MatchStatusDriverConfirmed
		// Match confirmed by driver, waiting for passenger
	} else if match.PassengerConfirmed {
		match.Status = matchmodels.MatchStatusPassengerConfirmed
		// Match confirmed by passenger, waiting for driver
	}

	match.UpdatedAt = time.Now()
	return uc.matchRepo.ConfirmMatchByUser(ctx, match.ID.String(), userID, isDriver)
}

// handleMatchAcceptance processes match acceptance logic
func (uc *MatchUC) handleMatchAcceptance(ctx context.Context, match *matchmodels.Match, req *matchmodels.MatchConfirmRequest) (matchmodels.MatchProposal, error) {
	isDriver := req.UserID == match.DriverID.String()

	updatedMatch, err := uc.updateMatchConfirmation(ctx, match, req.UserID, isDriver)
	if err != nil {
		logger.Warn("Failed to update match confirmation",
			logger.String("match_id", match.ID.String()),
			logger.ErrorField(err))
		updatedMatch = match // Use original match if update fails
	}

	// If match is fully accepted, handle auto-rejection asynchronously
	if updatedMatch.Status == matchmodels.MatchStatusAccepted {
		uc.startAsyncAutoRejection(updatedMatch)
		uc.PublishMatchAccepted(ctx, updatedMatch)
	}

	responseEvent := uc.buildMatchProposal(updatedMatch)
	// Created match proposal response

	return responseEvent, nil
}

func (uc *MatchUC) PublishMatchAccepted(ctx context.Context, match *matchmodels.Match) {
	// Create match proposal for accepted match
	PublishMatchAccepted := matchmodels.MatchProposal{
		ID:             match.ID.String(),
		PassengerID:    core.UUIDToStr(match.PassengerID),
		DriverID:       core.UUIDToStr(match.DriverID),
		UserLocation:   match.PassengerLocation,
		DriverLocation: match.DriverLocation,
		TargetLocation: match.TargetLocation,
		MatchStatus:    match.Status,
	}

	if err := uc.matchGW.PublishMatchAccepted(ctx, PublishMatchAccepted); err != nil {
		logger.Error("Failed to publish match accepted event",
			logger.String("match_id", PublishMatchAccepted.ID),
			logger.ErrorField(err))
	}
}

// startAsyncAutoRejection initiates the asynchronous auto-rejection process
func (uc *MatchUC) startAsyncAutoRejection(match *matchmodels.Match) {
	// Create context with timeout for background operation
	bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Start auto-rejection in background with proper context management
	go func() {
		defer cancel() // Ensure context is cleaned up

		if err := uc.handleAutoRejectionForAcceptedMatch(bgCtx, match); err != nil {
			logger.Error("Critical: Failed to handle auto-rejection for match",
				logger.String("match_id", match.ID.String()),
				logger.ErrorField(err))
		}
	}()
}

// handleAutoRejectionForAcceptedMatch rejects all other pending matches for the same passenger
func (uc *MatchUC) handleAutoRejectionForAcceptedMatch(ctx context.Context, acceptedMatch *matchmodels.Match) error {
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
	eventBatch := make([]matchmodels.MatchProposal, 0)

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
		if otherMatch.Status == matchmodels.MatchStatusPending ||
			otherMatch.Status == matchmodels.MatchStatusDriverConfirmed ||
			otherMatch.Status == matchmodels.MatchStatusPassengerConfirmed {

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
func (uc *MatchUC) processRejectionBatch(ctx context.Context, rejectionBatch []string, eventBatch []matchmodels.MatchProposal) error {
	if len(rejectionBatch) == 0 {
		return nil
	}

	// Attempt batch update
	if err := uc.matchRepo.BatchUpdateMatchStatus(ctx, rejectionBatch, matchmodels.MatchStatusRejected); err != nil {
		logger.Warn("Batch update failed, falling back to individual updates",
			logger.Int("batch_size", len(rejectionBatch)),
			logger.ErrorField(err))
		return uc.processIndividualRejections(ctx, rejectionBatch, eventBatch)
	}

	// Batch publish events
	return uc.publishRejectionEvents(ctx, eventBatch)
}

// processIndividualRejections handles individual updates when batch update fails
func (uc *MatchUC) processIndividualRejections(ctx context.Context, matchIDs []string, events []matchmodels.MatchProposal) error {
	for i, matchID := range matchIDs {
		select {
		case <-ctx.Done():
			return fmt.Errorf("auto-rejection cancelled during fallback updates: %w", ctx.Err())
		default:
		}

		if err := uc.matchRepo.UpdateMatchStatus(ctx, matchID, matchmodels.MatchStatusRejected); err != nil {
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
func (uc *MatchUC) publishRejectionEvents(ctx context.Context, events []matchmodels.MatchProposal) error {
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
func (uc *MatchUC) createRejectionEvent(match *matchmodels.Match) matchmodels.MatchProposal {
	return matchmodels.MatchProposal{
		ID:             match.ID.String(),
		PassengerID:    core.UUIDToStr(match.PassengerID),
		DriverID:       core.UUIDToStr(match.DriverID),
		MatchStatus:    matchmodels.MatchStatusRejected,
		DriverLocation: match.DriverLocation,
		UserLocation:   match.PassengerLocation,
		TargetLocation: match.TargetLocation,
	}
}

// handleMatchRejection processes match rejection logic
func (uc *MatchUC) handleMatchRejection(ctx context.Context, match *matchmodels.Match) (matchmodels.MatchProposal, error) {
	matchID := match.ID.String()

	if err := uc.matchRepo.UpdateMatchStatus(ctx, matchID, matchmodels.MatchStatusRejected); err != nil {
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
		match.Status = matchmodels.MatchStatusRejected // Fallback to local update
		updatedMatch = match
	}

	// Publish match rejection event
	matchProposal := uc.buildMatchProposal(updatedMatch)
	if err := uc.matchGW.PublishMatchRejected(ctx, matchProposal); err != nil {
		logger.Error("Failed to publish match rejection event",
			logger.String("match_id", matchID),
			logger.ErrorField(err))
	}

	return matchProposal, nil
}

// ConfirmMatchStatus handles match confirmation from either driver or passenger
func (uc *MatchUC) ConfirmMatchStatus(ctx context.Context, req *matchmodels.MatchConfirmRequest) (matchmodels.MatchProposal, error) {
	// Extract transaction from standard context
	txn := newrelic.FromContext(ctx)
	if txn != nil {
		segment := txn.StartSegment("MatchUC.ConfirmMatchStatus")
		defer segment.End()
	}

	// Get the match from database
	match, err := uc.matchRepo.GetMatch(ctx, req.ID)
	if err != nil {
		return matchmodels.MatchProposal{}, fmt.Errorf("match not found in database: %w", err)
	}

	switch req.Status {
	case string(matchmodels.MatchStatusAccepted):
		return uc.handleMatchAcceptance(ctx, match, req)
	case string(matchmodels.MatchStatusRejected):
		return uc.handleMatchRejection(ctx, match)
	default:
		err := fmt.Errorf("unsupported match status: %s", req.Status)
		return matchmodels.MatchProposal{}, err
	}
}

// GetMatch retrieves a match by ID
func (uc *MatchUC) GetMatch(ctx context.Context, matchID string) (*matchmodels.Match, error) {
	return uc.matchRepo.GetMatch(ctx, matchID)
}

// GetPendingMatch retrieves a pending match by ID
func (uc *MatchUC) GetPendingMatch(ctx context.Context, matchID string) (*matchmodels.Match, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to find match: %w", err)
	}

	// Only return if it's in pending state
	if match.Status == matchmodels.MatchStatusPending ||
		match.Status == matchmodels.MatchStatusDriverConfirmed ||
		match.Status == matchmodels.MatchStatusPassengerConfirmed {
		return match, nil
	}

	return nil, fmt.Errorf("match is not in pending state")
}

// RemoveDriverFromPool removes a driver from the available pool (locks them)
func (uc *MatchUC) RemoveDriverFromPool(ctx context.Context, driverID string) error {
	// Locking driver (removing from available pool)

	// Remove driver from available pool
	if err := uc.matchGW.RemoveAvailableDriver(ctx, driverID); err != nil {
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
	if err := uc.matchGW.RemoveAvailablePassenger(ctx, passengerID); err != nil {
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
