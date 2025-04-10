package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location"
	"github.com/piresc/nebengjek/services/match"
)

// Topics for NATS messaging
const (
	LocationUpdatesTopic   = "location_updates"
	MatchRequestsTopic     = "match_requests"
	MatchNotificationTopic = "match_notifications"
)

// LocationUpdate represents a driver location update message
type LocationUpdate struct {
	DriverID  string          `json:"driver_id"`
	Location  models.Location `json:"location"`
	Timestamp time.Time       `json:"timestamp"`
}

// MatchRequest represents a passenger match request message
type MatchRequest struct {
	PassengerID     string          `json:"passenger_id"`
	PickupLocation  models.Location `json:"pickup_location"`
	DropoffLocation models.Location `json:"dropoff_location"`
	Timestamp       time.Time       `json:"timestamp"`
}

// MatchNotification represents a match notification message
type MatchNotification struct {
	TripID      string    `json:"trip_id"`
	PassengerID string    `json:"passenger_id"`
	DriverID    string    `json:"driver_id"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

// MatchUC implements the match use case interface
type MatchUC struct {
	cfg          *models.Config
	repo         match.MatchRepo
	locationRepo location.LocationRepo
	natsProducer *nats.Producer
}

// NewMatchUseCase creates a new match use case
func NewMatchUseCase(
	cfg *models.Config,
	repo match.MatchRepo,
) *MatchUC {
	// Initialize NATS producer
	producer, err := nats.NewProducer(cfg.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to initialize NATS producer: %v", err)
	}

	return &MatchUC{
		cfg:          cfg,
		repo:         repo,
		natsProducer: producer,
	}
}

// InitConsumers initializes NATS consumers for location updates and match requests
func (uc *MatchUC) InitConsumers() error {
	// Initialize location updates consumer
	_, err := nats.NewConsumer(
		LocationUpdatesTopic,
		"match-service", // queue group
		uc.cfg.NATS.URL,
		uc.handleLocationUpdate,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize location updates consumer: %w", err)
	}

	// Initialize match requests consumer
	_, err = nats.NewConsumer(
		MatchRequestsTopic,
		"match-service", // queue group
		uc.cfg.NATS.URL,
		uc.handleMatchRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize match requests consumer: %w", err)
	}

	log.Println("NATS consumers initialized successfully")
	return nil
}

// handleLocationUpdate processes location update messages from NATS
func (uc *MatchUC) handleLocationUpdate(messageBody []byte) error {
	var update LocationUpdate
	err := json.Unmarshal(messageBody, &update)
	if err != nil {
		return fmt.Errorf("failed to unmarshal location update: %w", err)
	}

	log.Printf("Received location update for driver %s: %v", update.DriverID, update.Location)

	// Process the location update
	ctx := context.Background()
	return uc.ProcessLocationUpdate(ctx, update.DriverID, &update.Location)
}

// handleMatchRequest processes match request messages from NATS
func (uc *MatchUC) handleMatchRequest(messageBody []byte) error {
	var request MatchRequest
	err := json.Unmarshal(messageBody, &request)
	if err != nil {
		return fmt.Errorf("failed to unmarshal match request: %w", err)
	}

	log.Printf("Received match request from passenger %s", request.PassengerID)

	// Create a trip object from the request
	trip := &models.Trip{
		PassengerID:     request.PassengerID,
		PickupLocation:  request.PickupLocation,
		DropoffLocation: request.DropoffLocation,
		RequestedAt:     request.Timestamp,
		Status:          models.TripStatusRequested,
	}

	// Process the match request
	ctx := context.Background()
	return uc.CreateMatchRequest(ctx, trip)
}

// CreateMatchRequest creates a new match request
func (uc *MatchUC) CreateMatchRequest(ctx context.Context, trip *models.Trip) error {
	// Save the trip to the database
	err := uc.repo.CreateMatch(ctx, trip)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	// Find a suitable driver for the match
	matchedTrip, err := uc.FindMatchForPassenger(ctx, trip.PassengerID, &trip.PickupLocation)
	if err != nil {
		log.Printf("No immediate match found for passenger %s: %v", trip.PassengerID, err)
		// This is not a critical error, as we might find a match later
		return nil
	}

	// If a match was found, update the trip status and notify
	if matchedTrip != nil && matchedTrip.DriverID != "" {
		// Update the trip status to matched
		err = uc.repo.UpdateMatchStatus(ctx, matchedTrip.ID, models.TripStatusMatched)
		if err != nil {
			return fmt.Errorf("failed to update match status: %w", err)
		}

		// Send notification about the match
		uc.sendMatchNotification(matchedTrip)
	}

	return nil
}

// ProcessLocationUpdate processes a driver location update
func (uc *MatchUC) ProcessLocationUpdate(ctx context.Context, driverID string, location *models.Location) error {
	// Update the driver's location in the database
	err := uc.locationRepo.UpdateDriverLocation(ctx, driverID, location)
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}

	// Check for pending match requests that could be matched with this driver
	// This is a simplified approach - in a real system, you might use a more sophisticated matching algorithm
	// or a separate background process for matching

	// Get pending match requests (trips with status REQUESTED)
	// For simplicity, we're not implementing this here, but in a real system,
	// you would query for pending requests and try to match them with the driver

	return nil
}

// FindMatchForPassenger finds a suitable driver for a passenger
func (uc *MatchUC) FindMatchForPassenger(ctx context.Context, passengerID string, location *models.Location) (*models.Trip, error) {
	// Find nearby available drivers within 1 km radius
	drivers, err := uc.locationRepo.GetNearbyDrivers(ctx, location, 1.0) // 1.0 km radius
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	if len(drivers) == 0 {
		return nil, fmt.Errorf("no available drivers found nearby")
	}

	// For simplicity, we'll just pick the first available driver
	// In a real system, you would use a more sophisticated matching algorithm
	// that considers factors like driver rating, ETA, etc.
	selectedDriver := drivers[0]

	// Get the pending trip for this passenger
	pendingTrips, err := uc.repo.GetPendingMatchesByPassengerID(ctx, passengerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending trips: %w", err)
	}

	if len(pendingTrips) == 0 {
		return nil, fmt.Errorf("no pending trip found for passenger")
	}

	// Get the most recent pending trip
	trip := pendingTrips[0]

	// Update the trip with the selected driver
	trip.DriverID = selectedDriver.ID
	trip.Status = models.TripStatusMatched
	trip.MatchedAt = timePtr(time.Now())

	// Update the trip in the database
	err = uc.repo.UpdateMatchStatus(ctx, trip.ID, models.TripStatusMatched)
	if err != nil {
		return nil, fmt.Errorf("failed to update match status: %w", err)
	}

	return trip, nil
}

// AcceptMatch allows a driver to accept a match
func (uc *MatchUC) AcceptMatch(ctx context.Context, tripID string, driverID string) error {
	// Get the trip
	trip, err := uc.repo.GetMatchByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Verify that the driver is assigned to this trip
	if trip.DriverID != driverID {
		return fmt.Errorf("driver is not assigned to this trip")
	}

	// Verify that the trip is in MATCHED status
	if trip.Status != models.TripStatusMatched {
		return fmt.Errorf("trip is not in MATCHED status")
	}

	// Update the trip status to ACCEPTED
	err = uc.repo.UpdateMatchStatus(ctx, tripID, models.TripStatusAccepted)
	if err != nil {
		return fmt.Errorf("failed to update match status: %w", err)
	}

	// Send notification about the acceptance
	trip.Status = models.TripStatusAccepted
	uc.sendMatchNotification(trip)

	return nil
}

// RejectMatch allows a driver to reject a match
func (uc *MatchUC) RejectMatch(ctx context.Context, tripID string, driverID string) error {
	// Get the trip
	trip, err := uc.repo.GetMatchByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Verify that the driver is assigned to this trip
	if trip.DriverID != driverID {
		return fmt.Errorf("driver is not assigned to this trip")
	}

	// Verify that the trip is in MATCHED status
	if trip.Status != models.TripStatusMatched {
		return fmt.Errorf("trip is not in MATCHED status")
	}

	// Update the trip status to REJECTED
	err = uc.repo.UpdateMatchStatus(ctx, tripID, models.TripStatusRejected)
	if err != nil {
		return fmt.Errorf("failed to update match status: %w", err)
	}

	// Send notification about the rejection
	trip.Status = models.TripStatusRejected
	uc.sendMatchNotification(trip)

	// Try to find another driver for this passenger
	go func() {
		ctx := context.Background()
		_, err := uc.FindMatchForPassenger(ctx, trip.PassengerID, &trip.PickupLocation)
		if err != nil {
			log.Printf("Failed to find new match after rejection: %v", err)
		}
	}()

	return nil
}

// CancelMatch allows a user (passenger or driver) to cancel a match
func (uc *MatchUC) CancelMatch(ctx context.Context, tripID string, userID string) error {
	// Get the trip
	trip, err := uc.repo.GetMatchByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Verify that the user is associated with this trip
	if trip.PassengerID != userID && trip.DriverID != userID {
		return fmt.Errorf("user is not associated with this trip")
	}

	// Verify that the trip is in a cancellable status
	if trip.Status != models.TripStatusRequested &&
		trip.Status != models.TripStatusMatched &&
		trip.Status != models.TripStatusAccepted {
		return fmt.Errorf("trip cannot be cancelled in its current status")
	}

	// Update the trip status to CANCELLED
	err = uc.repo.UpdateMatchStatus(ctx, tripID, models.TripStatusCancelled)
	if err != nil {
		return fmt.Errorf("failed to update match status: %w", err)
	}

	// Send notification about the cancellation
	trip.Status = models.TripStatusCancelled
	uc.sendMatchNotification(trip)

	return nil
}

// GetPendingMatchesForDriver retrieves pending matches for a driver
func (uc *MatchUC) GetPendingMatchesForDriver(ctx context.Context, driverID string) ([]*models.Trip, error) {
	return uc.repo.GetPendingMatchesByDriverID(ctx, driverID)
}

// GetPendingMatchesForPassenger retrieves pending matches for a passenger
func (uc *MatchUC) GetPendingMatchesForPassenger(ctx context.Context, passengerID string) ([]*models.Trip, error) {
	return uc.repo.GetPendingMatchesByPassengerID(ctx, passengerID)
}

// sendMatchNotification sends a notification about a match event
func (uc *MatchUC) sendMatchNotification(trip *models.Trip) {
	notification := MatchNotification{
		TripID:      trip.ID,
		PassengerID: trip.PassengerID,
		DriverID:    trip.DriverID,
		Status:      string(trip.Status),
		Timestamp:   time.Now(),
	}

	err := uc.natsProducer.Publish(MatchNotificationTopic, notification)
	if err != nil {
		log.Printf("Failed to send match notification: %v", err)
	}
}

// Helper function to create a time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
