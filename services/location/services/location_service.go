package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location"
)

// Topics for NATS messaging
const (
	LocationUpdatesTopic = "location_updates"
)

// LocationUpdateType defines the type of location update
type LocationUpdateType string

const (
	// PeriodicUpdate is sent every 30-60 seconds in the background
	PeriodicUpdate LocationUpdateType = "periodic"
	// EventBasedUpdate is sent when app state changes or user initiates a trip request
	EventBasedUpdate LocationUpdateType = "event_based"
)

// LocationUpdate represents a location update message
type LocationUpdate struct {
	UserID     string             `json:"user_id"`
	Role       string             `json:"role"` // driver or customer
	Location   models.Location    `json:"location"`
	Timestamp  time.Time          `json:"timestamp"`
	UpdateType LocationUpdateType `json:"update_type"`
	EventType  string             `json:"event_type,omitempty"` // For event-based updates: app_state_change, trip_request, etc.
}

// LocationService implements location service functionality
type LocationService struct {
	cfg          *models.Config
	locationRepo location.LocationRepo
	redisClient  *database.RedisClient
	natsProducer *nats.Producer
	workers      map[string]*locationWorker // Map of user ID to worker
	mu           sync.Mutex                 // Mutex for workers map
}

// locationWorker handles periodic location updates for a specific user
type locationWorker struct {
	userID       string
	role         string
	interval     time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	locationRepo location.LocationRepo
	redisClient  *database.RedisClient
	natsProducer *nats.Producer
}

// NewLocationService creates a new location service
func NewLocationService(
	cfg *models.Config,
	locationRepo location.LocationRepo,
	redisClient *database.RedisClient,
) (*LocationService, error) {
	// Initialize NATS producer
	producer, err := nats.NewProducer(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NATS producer: %w", err)
	}

	return &LocationService{
		cfg:          cfg,
		locationRepo: locationRepo,
		redisClient:  redisClient,
		natsProducer: producer,
		workers:      make(map[string]*locationWorker),
	}, nil
}

// StartPeriodicUpdates starts periodic location updates for a user
func (s *LocationService) StartPeriodicUpdates(userID, role string, interval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if worker already exists
	if _, exists := s.workers[userID]; exists {
		return fmt.Errorf("periodic updates already running for user %s", userID)
	}

	// Create a new worker
	ctx, cancel := context.WithCancel(context.Background())
	worker := &locationWorker{
		userID:       userID,
		role:         role,
		interval:     interval,
		ctx:          ctx,
		cancel:       cancel,
		locationRepo: s.locationRepo,
		redisClient:  s.redisClient,
		natsProducer: s.natsProducer,
	}

	// Start the worker
	go worker.run()

	// Add worker to map
	s.workers[userID] = worker

	log.Printf("Started periodic location updates for user %s with interval %v", userID, interval)
	return nil
}

// StopPeriodicUpdates stops periodic location updates for a user
func (s *LocationService) StopPeriodicUpdates(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if worker exists
	worker, exists := s.workers[userID]
	if !exists {
		return fmt.Errorf("no periodic updates running for user %s", userID)
	}

	// Stop the worker
	worker.cancel()

	// Remove worker from map
	delete(s.workers, userID)

	log.Printf("Stopped periodic location updates for user %s", userID)
	return nil
}

// UpdateLocation updates a user's location and publishes it to NATS
func (s *LocationService) UpdateLocation(
	ctx context.Context,
	userID string,
	role string,
	location *models.Location,
	updateType LocationUpdateType,
	eventType string,
) error {
	// Validate location
	if location == nil {
		return fmt.Errorf("location cannot be nil")
	}

	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Update location in database based on role
	var err error
	if role == "driver" {
		err = s.locationRepo.UpdateDriverLocation(ctx, userID, location)
	} else {
		// For future implementation: customer location updates
		// err = s.locationRepo.UpdateCustomerLocation(ctx, userID, location)
		return fmt.Errorf("customer location updates not implemented yet")
	}

	if err != nil {
		return fmt.Errorf("failed to update location in database: %w", err)
	}

	// Update location in Redis for quick geospatial queries
	if role == "driver" {
		err = s.redisClient.GeoAdd(
			ctx,
			constants.DriverLocationKey,
			location.Longitude,
			location.Latitude,
			userID,
		)
		if err != nil {
			log.Printf("Warning: failed to update location in Redis: %v", err)
			// Continue even if Redis update fails
		}
	}

	// Publish location update to NATS
	update := LocationUpdate{
		UserID:     userID,
		Role:       role,
		Location:   *location,
		Timestamp:  location.Timestamp,
		UpdateType: updateType,
		EventType:  eventType,
	}

	err = s.natsProducer.Publish(LocationUpdatesTopic, update)
	if err != nil {
		log.Printf("Warning: failed to publish location update to NATS: %v", err)
		// Continue even if NATS publish fails
	}

	return nil
}

// GetNearbyDriversFromRedis gets nearby drivers using Redis geospatial queries
func (s *LocationService) GetNearbyDriversFromRedis(
	ctx context.Context,
	location *models.Location,
	radiusKm float64,
) ([]string, error) {
	// Get nearby drivers from Redis
	results, err := s.redisClient.GeoRadius(
		ctx,
		constants.DriverLocationKey,
		location.Longitude,
		location.Latitude,
		radiusKm,
		"km",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get nearby drivers from Redis: %w", err)
	}

	// Extract driver IDs
	driverIDs := make([]string, len(results))
	for i, result := range results {
		driverIDs[i] = result.Name
	}

	return driverIDs, nil
}

// run is the worker's main loop for periodic location updates
func (w *locationWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get the latest location for this user
			// In a real implementation, this would come from a mobile device
			// For this example, we'll just log that we would update the location
			log.Printf("Would update location for user %s (periodic update)", w.userID)

			// In a real implementation with actual location data:
			// location := getCurrentLocationFromDevice()
			// w.locationRepo.UpdateDriverLocation(context.Background(), w.userID, location)
			// w.redisClient.GeoAdd(...)
			// w.natsProducer.Publish(...)

		case <-w.ctx.Done():
			log.Printf("Stopping periodic location updates for user %s", w.userID)
			return
		}
	}
}
