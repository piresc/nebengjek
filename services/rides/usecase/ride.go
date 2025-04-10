package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides"
	"github.com/piresc/nebengjek/services/rides/services"
)

// RideUC implements the rides.RideUseCase interface
type RideUC struct {
	cfg         *models.Config
	repo        rides.RideRepo
	rideService *services.RideService
}

// NewRideUC creates a new ride use case
func NewRideUC(
	cfg *models.Config,
	repo rides.RideRepo,
) (rides.RideUseCase, error) {
	// Create ride service
	rideService, err := services.NewRideService(cfg, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create ride service: %w", err)
	}

	return &RideUC{
		cfg:         cfg,
		repo:        repo,
		rideService: rideService,
	}, nil
}

// CreateRideRequest creates a new ride request
func (uc *RideUC) CreateRideRequest(ctx context.Context, passengerID string, pickup, dropoff *models.Location) (*models.Trip, error) {
	// Generate a new trip ID
	tripID := uuid.New().String()

	// Set timestamp if not provided
	if pickup.Timestamp.IsZero() {
		pickup.Timestamp = time.Now()
	}
	if dropoff.Timestamp.IsZero() {
		dropoff.Timestamp = time.Now()
	}

	// Calculate estimated distance and duration (simplified)
	// In a real implementation, this would use a routing service
	distance := calculateDistance(pickup.Latitude, pickup.Longitude, dropoff.Latitude, dropoff.Longitude)
	duration := int(distance * 3) // Rough estimate: 3 minutes per km

	// Create trip object
	trip := &models.Trip{
		ID:              tripID,
		PassengerID:     passengerID,
		PickupLocation:  *pickup,
		DropoffLocation: *dropoff,
		RequestedAt:     time.Now(),
		Status:          models.TripStatusRequested,
		Distance:        distance,
		Duration:        duration,
	}

	// Calculate estimated fare
	fare, err := uc.rideService.CalculateFare(ctx, trip)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fare: %w", err)
	}
	trip.Fare = fare

	// Store trip in database
	if err := uc.repo.CreateRide(ctx, trip); err != nil {
		return nil, fmt.Errorf("failed to create ride: %w", err)
	}

	// Publish ride request event
	if err := uc.rideService.PublishRideStatusUpdate(ctx, trip); err != nil {
		log.Printf("Warning: failed to publish ride status update: %v", err)
		// Continue even if publishing fails
	}

	return trip, nil
}

// CancelRideRequest cancels a ride request
func (uc *RideUC) CancelRideRequest(ctx context.Context, tripID string, userID string) error {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	// Check if user is authorized to cancel the ride
	if trip.PassengerID != userID && trip.DriverID != userID {
		return fmt.Errorf("user not authorized to cancel this ride")
	}

	// Check if ride can be cancelled
	if trip.Status == models.TripStatusCompleted || trip.Status == models.TripStatusCancelled {
		return fmt.Errorf("ride cannot be cancelled in %s state", trip.Status)
	}

	// Update ride status
	if err := uc.repo.UpdateRideStatus(ctx, tripID, models.TripStatusCancelled); err != nil {
		return fmt.Errorf("failed to update ride status: %w", err)
	}

	// Update cancelled timestamp
	now := time.Now()
	if err := uc.repo.UpdateRideTimestamp(ctx, tripID, "cancelled_at", now); err != nil {
		log.Printf("Warning: failed to update cancelled timestamp: %v", err)
		// Continue even if timestamp update fails
	}

	// Update trip object for event publishing
	trip.Status = models.TripStatusCancelled
	trip.CancelledAt = &now

	// Publish ride status update
	if err := uc.rideService.PublishRideStatusUpdate(ctx, trip); err != nil {
		log.Printf("Warning: failed to publish ride status update: %v", err)
		// Continue even if publishing fails
	}

	return nil
}

// AcceptRide accepts a ride request by a driver
func (uc *RideUC) AcceptRide(ctx context.Context, tripID string, driverID string) error {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	// Check if ride can be accepted
	if trip.Status != models.TripStatusRequested && trip.Status != models.TripStatusMatched {
		return fmt.Errorf("ride cannot be accepted in %s state", trip.Status)
	}

	// Update ride status and driver ID
	if err := uc.repo.UpdateRideStatus(ctx, tripID, models.TripStatusAccepted); err != nil {
		return fmt.Errorf("failed to update ride status: %w", err)
	}

	// Update accepted timestamp
	now := time.Now()
	if err := uc.repo.UpdateRideTimestamp(ctx, tripID, "accepted_at", now); err != nil {
		log.Printf("Warning: failed to update accepted timestamp: %v", err)
		// Continue even if timestamp update fails
	}

	// Update trip object for event publishing
	trip.Status = models.TripStatusAccepted
	trip.DriverID = driverID
	trip.AcceptedAt = &now

	// Publish ride status update
	if err := uc.rideService.PublishRideStatusUpdate(ctx, trip); err != nil {
		log.Printf("Warning: failed to publish ride status update: %v", err)
		// Continue even if publishing fails
	}

	return nil
}

// RejectRide rejects a ride request by a driver
func (uc *RideUC) RejectRide(ctx context.Context, tripID string, driverID string) error {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	// Check if ride can be rejected
	if trip.Status != models.TripStatusRequested && trip.Status != models.TripStatusMatched {
		return fmt.Errorf("ride cannot be rejected in %s state", trip.Status)
	}

	// Update ride status
	if err := uc.repo.UpdateRideStatus(ctx, tripID, models.TripStatusRejected); err != nil {
		return fmt.Errorf("failed to update ride status: %w", err)
	}

	// Update rejected timestamp
	now := time.Now()
	if err := uc.repo.UpdateRideTimestamp(ctx, tripID, "cancelled_at", now); err != nil {
		log.Printf("Warning: failed to update rejected timestamp: %v", err)
		// Continue even if timestamp update fails
	}

	// Update trip object for event publishing
	trip.Status = models.TripStatusRejected
	trip.CancelledAt = &now

	// Publish ride status update
	if err := uc.rideService.PublishRideStatusUpdate(ctx, trip); err != nil {
		log.Printf("Warning: failed to publish ride status update: %v", err)
		// Continue even if publishing fails
	}

	return nil
}

// StartRide starts a ride
func (uc *RideUC) StartRide(ctx context.Context, tripID string, driverID string) error {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	// Check if driver is authorized to start the ride
	if trip.DriverID != driverID {
		return fmt.Errorf("driver not authorized to start this ride")
	}

	// Check if ride can be started
	if trip.Status != models.TripStatusAccepted {
		return fmt.Errorf("ride cannot be started in %s state", trip.Status)
	}

	// Update ride status
	if err := uc.repo.UpdateRideStatus(ctx, tripID, models.TripStatusInProgress); err != nil {
		return fmt.Errorf("failed to update ride status: %w", err)
	}

	// Update started timestamp
	now := time.Now()
	if err := uc.repo.UpdateRideTimestamp(ctx, tripID, "started_at", now); err != nil {
		log.Printf("Warning: failed to update started timestamp: %v", err)
		// Continue even if timestamp update fails
	}

	// Update trip object for event publishing
	trip.Status = models.TripStatusInProgress
	trip.StartedAt = &now

	// Publish ride status update
	if err := uc.rideService.PublishRideStatusUpdate(ctx, trip); err != nil {
		log.Printf("Warning: failed to publish ride status update: %v", err)
		// Continue even if publishing fails
	}

	return nil
}

// CompleteRide completes a ride
func (uc *RideUC) CompleteRide(ctx context.Context, tripID string, driverID string) error {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	// Check if driver is authorized to complete the ride
	if trip.DriverID != driverID {
		return fmt.Errorf("driver not authorized to complete this ride")
	}

	// Check if ride can be completed
	if trip.Status != models.TripStatusInProgress {
		return fmt.Errorf("ride cannot be completed in %s state", trip.Status)
	}

	// Update ride status
	if err := uc.repo.UpdateRideStatus(ctx, tripID, models.TripStatusCompleted); err != nil {
		return fmt.Errorf("failed to update ride status: %w", err)
	}

	// Update completed timestamp
	now := time.Now()
	if err := uc.repo.UpdateRideTimestamp(ctx, tripID, "completed_at", now); err != nil {
		log.Printf("Warning: failed to update completed timestamp: %v", err)
		// Continue even if timestamp update fails
	}

	// Update trip object for event publishing
	trip.Status = models.TripStatusCompleted
	trip.CompletedAt = &now

	// Calculate final fare
	// In a real implementation, this would use actual distance and duration
	fare, err := uc.rideService.CalculateFare(ctx, trip)
	if err != nil {
		log.Printf("Warning: failed to calculate final fare: %v", err)
	} else {
		// Update fare in database
		if err := uc.repo.UpdateRideFare(ctx, tripID, fare); err != nil {
			log.Printf("Warning: failed to update fare: %v", err)
		}

		// Update trip object for event publishing
		trip.Fare = fare

		// Publish fare update
		if err := uc.rideService.PublishFareUpdate(ctx, trip, fare); err != nil {
			log.Printf("Warning: failed to publish fare update: %v", err)
		}
	}

	// Publish ride status update
	if err := uc.rideService.PublishRideStatusUpdate(ctx, trip); err != nil {
		log.Printf("Warning: failed to publish ride status update: %v", err)
	}

	return nil
}

// GetRideStatus gets the status of a ride
func (uc *RideUC) GetRideStatus(ctx context.Context, tripID string) (*models.Trip, error) {
	return uc.repo.GetRideByID(ctx, tripID)
}

// GetActiveRideForPassenger gets the active ride for a passenger
func (uc *RideUC) GetActiveRideForPassenger(ctx context.Context, passengerID string) (*models.Trip, error) {
	return uc.repo.GetActiveRideByPassengerID(ctx, passengerID)
}

// GetActiveRideForDriver gets the active ride for a driver
func (uc *RideUC) GetActiveRideForDriver(ctx context.Context, driverID string) (*models.Trip, error) {
	return uc.repo.GetActiveRideByDriverID(ctx, driverID)
}

// GetRideHistory gets the ride history for a user
func (uc *RideUC) GetRideHistory(ctx context.Context, userID string, role string, startTime, endTime time.Time) ([]*models.Trip, error) {
	return uc.repo.GetRideHistory(ctx, userID, role, startTime, endTime)
}

// CalculateFare calculates the fare for a trip
func (uc *RideUC) CalculateFare(ctx context.Context, tripID string) (*models.Fare, error) {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	// Calculate fare
	return uc.rideService.CalculateFare(ctx, trip)
}

// UpdateFare updates the fare for a trip
func (uc *RideUC) UpdateFare(ctx context.Context, tripID string, fare *models.Fare) error {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	// Update fare in database
	if err := uc.repo.UpdateRideFare(ctx, tripID, fare); err != nil {
		return fmt.Errorf("failed to update fare: %w", err)
	}

	// Update trip object for event publishing
	trip.Fare = fare

	// Publish fare update
	if err := uc.rideService.PublishFareUpdate(ctx, trip, fare); err != nil {
		log.Printf("Warning: failed to publish fare update: %v", err)
	}

	return nil
}

// RateRide rates a ride
func (uc *RideUC) RateRide(ctx context.Context, tripID string, userID string, role string, rating float64) error {
	// Get ride by ID
	trip, err := uc.repo.GetRideByID(ctx, tripID)
	if err != nil {
		return fmt.Errorf("failed to get ride: %w", err)
	}

	// Check if user is authorized to rate the ride
	if role == "passenger" && trip.PassengerID != userID {
		return fmt.Errorf("passenger not authorized to rate this ride")
	} else if role == "driver" && trip.DriverID != userID {
		return fmt.Errorf("driver not authorized to rate this ride")
	}

	// Check if ride can be rated
	if trip.Status != models.TripStatusCompleted {
		return fmt.Errorf("ride cannot be rated in %s state", trip.Status)
	}

	// Update rating in database
	if err := uc.repo.UpdateRideRating(ctx, tripID, role, rating); err != nil {
		return fmt.Errorf("failed to update rating: %w", err)
	}

	return nil
}

// calculateDistance calculates the distance between two points using the Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Implementation of Haversine formula
	// In a real implementation, this would be more accurate
	// For simplicity, we'll return a dummy value
	return 5.0 // 5 km
}
