package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location"
	"github.com/piresc/nebengjek/services/location/services"
)

// LocationUC implements the location.LocationUseCase interface
type LocationUC struct {
	repo         location.LocationRepo
	redisClient  *database.RedisClient
	locationSvc  *services.LocationService
	workers      map[string]context.CancelFunc
	workersMutex sync.Mutex
}

// NewLocationUC creates a new location use case
func NewLocationUC(repo location.LocationRepo, redisClient *database.RedisClient, cfg *models.Config) (location.LocationUseCase, error) {
	// Create location service
	locationSvc, err := services.NewLocationService(cfg, repo, redisClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create location service: %w", err)
	}

	return &LocationUC{
		repo:        repo,
		redisClient: redisClient,
		locationSvc: locationSvc,
		workers:     make(map[string]context.CancelFunc),
	}, nil
}

// UpdateDriverLocation updates a driver's current location
func (s *LocationUC) UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error {
	// Validate location data
	if err := validateLocationData(location); err != nil {
		return err
	}

	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Update driver location
	err := s.repo.UpdateDriverLocation(ctx, driverID, location)
	if err != nil {
		return err
	}

	// Store in location history
	err = s.repo.StoreLocationHistory(ctx, driverID, "driver", location)
	if err != nil {
		log.Printf("Warning: failed to store location history: %v", err)
		// Continue even if history storage fails
	}

	// Update location in the location service (which handles Redis and NATS)
	return s.locationSvc.UpdateLocation(ctx, driverID, "driver", location, services.EventBasedUpdate, "manual_update")
}

// UpdateDriverAvailability updates a driver's availability status
func (s *LocationUC) UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error {
	// Update driver availability
	return s.repo.UpdateDriverAvailability(ctx, driverID, isAvailable)
}

// GetNearbyDrivers retrieves available drivers near a location within a radius
func (s *LocationUC) GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error) {
	// Validate location data
	if err := validateLocationData(location); err != nil {
		return nil, err
	}

	// Validate radius
	if radiusKm <= 0 {
		return nil, fmt.Errorf("radius must be positive")
	}

	// Try to get nearby drivers from Redis first (faster)
	driverIDs, err := s.locationSvc.GetNearbyDriversFromRedis(ctx, location, radiusKm)
	if err == nil && len(driverIDs) > 0 {
		// Redis lookup successful, get driver details from database
		// This would require a new repository method to get drivers by IDs
		// For now, we'll fall back to the database query
	}

	// Fall back to database query if Redis lookup fails or returns no results
	drivers, err := s.repo.GetNearbyDrivers(ctx, location, radiusKm)
	if err != nil {
		return nil, err
	}
	return drivers, nil
}

// UpdateCustomerLocation updates a customer's current location
func (s *LocationUC) UpdateCustomerLocation(ctx context.Context, customerID string, location *models.Location) error {
	// Validate location data
	if err := validateLocationData(location); err != nil {
		return err
	}

	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Update customer location
	err := s.repo.UpdateCustomerLocation(ctx, customerID, location)
	if err != nil {
		return err
	}

	// Store in location history
	err = s.repo.StoreLocationHistory(ctx, customerID, "customer", location)
	if err != nil {
		log.Printf("Warning: failed to store location history: %v", err)
		// Continue even if history storage fails
	}

	// Update location in the location service (which handles Redis and NATS)
	return s.locationSvc.UpdateLocation(ctx, customerID, "customer", location, services.EventBasedUpdate, "manual_update")
}

// StartPeriodicUpdates starts periodic location updates for a user
func (s *LocationUC) StartPeriodicUpdates(ctx context.Context, userID string, role string, interval time.Duration) error {
	// Validate interval
	if interval < 30*time.Second {
		// Minimum interval is 30 seconds to avoid excessive updates
		interval = 30 * time.Second
	} else if interval > 60*time.Second {
		// Maximum interval is 60 seconds to ensure timely updates
		interval = 60 * time.Second
	}

	// Check if updates are already running for this user
	s.workersMutex.Lock()
	defer s.workersMutex.Unlock()

	if _, exists := s.workers[userID]; exists {
		return fmt.Errorf("periodic updates already running for user %s", userID)
	}

	// Start periodic updates in the location service
	err := s.locationSvc.StartPeriodicUpdates(userID, role, interval)
	if err != nil {
		return err
	}

	// Create a context with cancel function for this worker
	workerCtx, cancel := context.WithCancel(context.Background())
	s.workers[userID] = cancel

	// Start a goroutine to monitor the context
	go func() {
		<-workerCtx.Done()
		// Context was canceled, clean up
		log.Printf("Periodic updates for user %s stopped", userID)
	}()

	return nil
}

// StopPeriodicUpdates stops periodic location updates for a user
func (s *LocationUC) StopPeriodicUpdates(ctx context.Context, userID string) error {
	// Check if updates are running for this user
	s.workersMutex.Lock()
	defer s.workersMutex.Unlock()

	cancel, exists := s.workers[userID]
	if !exists {
		return fmt.Errorf("no periodic updates running for user %s", userID)
	}

	// Cancel the context to stop the worker
	cancel()

	// Remove the worker from the map
	delete(s.workers, userID)

	// Stop periodic updates in the location service
	return s.locationSvc.StopPeriodicUpdates(userID)
}

// UpdateLocationOnEvent updates a user's location based on an event
func (s *LocationUC) UpdateLocationOnEvent(ctx context.Context, userID string, role string, location *models.Location, eventType string) error {
	// Validate location data
	if err := validateLocationData(location); err != nil {
		return err
	}

	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Update location based on role
	var err error
	if role == "driver" {
		err = s.repo.UpdateDriverLocation(ctx, userID, location)
	} else if role == "customer" {
		err = s.repo.UpdateCustomerLocation(ctx, userID, location)
	} else {
		return fmt.Errorf("invalid role: %s", role)
	}

	if err != nil {
		return err
	}

	// Store in location history
	err = s.repo.StoreLocationHistory(ctx, userID, role, location)
	if err != nil {
		log.Printf("Warning: failed to store location history: %v", err)
		// Continue even if history storage fails
	}

	// Update location in the location service (which handles Redis and NATS)
	return s.locationSvc.UpdateLocation(ctx, userID, role, location, services.EventBasedUpdate, eventType)
}

// GetLocationHistory retrieves location history for a user within a time range
func (s *LocationUC) GetLocationHistory(ctx context.Context, userID string, startTime, endTime time.Time) ([]*models.Location, error) {
	// Validate time range
	if startTime.After(endTime) {
		return nil, fmt.Errorf("start time must be before end time")
	}

	// Get location history from repository
	return s.repo.GetLocationHistory(ctx, userID, startTime, endTime)
}

// Helper functions for validation

func validateLocationData(location *models.Location) error {
	if location == nil {
		return errors.New("location cannot be nil")
	}

	// Validate latitude (between -90 and 90)
	if location.Latitude < -90 || location.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}

	// Validate longitude (between -180 and 180)
	if location.Longitude < -180 || location.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}

	return nil
}
