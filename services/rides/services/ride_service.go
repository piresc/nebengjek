package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides"
)

// Topics for NATS messaging
const (
	RideUpdatesTopic = "ride_updates"
	RideRequestTopic = "ride_requests"
	RideStatusTopic  = "ride_status"
)

// RideUpdateType defines the type of ride update
type RideUpdateType string

const (
	// StatusUpdate is sent when ride status changes
	StatusUpdate RideUpdateType = "status_update"
	// LocationUpdate is sent when driver/passenger location changes
	LocationUpdate RideUpdateType = "location_update"
	// FareUpdate is sent when fare is calculated or updated
	FareUpdate RideUpdateType = "fare_update"
)

// RideUpdate represents a ride update message
type RideUpdate struct {
	TripID      string         `json:"trip_id"`
	UpdateType  RideUpdateType `json:"update_type"`
	Status      string         `json:"status,omitempty"`
	DriverID    string         `json:"driver_id,omitempty"`
	PassengerID string         `json:"passenger_id,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
	Fare        *models.Fare   `json:"fare,omitempty"`
}

// RideService implements ride service functionality
type RideService struct {
	cfg          *models.Config
	rideRepo     rides.RideRepo
	redisClient  *database.RedisClient
	natsProducer *nats.Producer
}

// NewRideService creates a new ride service
func NewRideService(
	cfg *models.Config,
	rideRepo rides.RideRepo,
) (*RideService, error) {
	// Initialize Redis client
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis client: %w", err)
	}

	// Initialize NATS producer
	producer, err := nats.NewProducer(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NATS producer: %w", err)
	}

	return &RideService{
		cfg:          cfg,
		rideRepo:     rideRepo,
		redisClient:  redisClient,
		natsProducer: producer,
	}, nil
}

// CalculateFare calculates the fare for a trip
func (s *RideService) CalculateFare(ctx context.Context, trip *models.Trip) (*models.Fare, error) {
	// Get base fare from config
	baseFare := s.cfg.Pricing.BaseFare
	if baseFare <= 0 {
		baseFare = 5.0 // Default base fare
	}

	// Get per km and per minute rates from config
	perKmRate := s.cfg.Pricing.PerKmRate
	if perKmRate <= 0 {
		perKmRate = 1.5 // Default per km rate
	}

	perMinuteRate := s.cfg.Pricing.PerMinuteRate
	if perMinuteRate <= 0 {
		perMinuteRate = 0.2 // Default per minute rate
	}

	// Calculate distance fare
	distanceFare := trip.Distance * perKmRate

	// Calculate duration fare
	durationFare := float64(trip.Duration) * perMinuteRate

	// Get surge factor from config or calculate based on demand
	surgeFactor := s.cfg.Pricing.SurgeFactor
	if surgeFactor < 1.0 {
		surgeFactor = 1.0 // Minimum surge factor
	}

	// Calculate total fare
	totalFare := (baseFare + distanceFare + durationFare) * surgeFactor

	// Round to 2 decimal places
	totalFare = math.Round(totalFare*100) / 100

	// Create fare object
	fare := &models.Fare{
		BaseFare:     baseFare,
		DistanceFare: distanceFare,
		DurationFare: durationFare,
		SurgeFactor:  surgeFactor,
		TotalFare:    totalFare,
		Currency:     s.cfg.Pricing.Currency,
	}

	return fare, nil
}

// PublishRideUpdate publishes a ride update to NATS
func (s *RideService) PublishRideUpdate(ctx context.Context, update *RideUpdate) error {
	// Store trip status in Redis for quick access
	if update.Status != "" {
		err := s.redisClient.Set(ctx, fmt.Sprintf(constants.KeyActiveTrip, update.TripID), update.Status, 24*time.Hour)
		if err != nil {
			log.Printf("Warning: failed to store trip status in Redis: %v", err)
		}
	}

	return s.natsProducer.Publish(RideUpdatesTopic, update)
}

// PublishRideStatusUpdate publishes a ride status update to NATS
func (s *RideService) PublishRideStatusUpdate(ctx context.Context, trip *models.Trip) error {
	update := &RideUpdate{
		TripID:      trip.ID,
		UpdateType:  StatusUpdate,
		Status:      string(trip.Status),
		DriverID:    trip.DriverID,
		PassengerID: trip.PassengerID,
		Timestamp:   time.Now(),
	}

	return s.PublishRideUpdate(ctx, update)
}

// PublishFareUpdate publishes a fare update to NATS
func (s *RideService) PublishFareUpdate(ctx context.Context, trip *models.Trip, fare *models.Fare) error {
	update := &RideUpdate{
		TripID:      trip.ID,
		UpdateType:  FareUpdate,
		DriverID:    trip.DriverID,
		PassengerID: trip.PassengerID,
		Timestamp:   time.Now(),
		Fare:        fare,
	}

	return s.PublishRideUpdate(ctx, update)
}

// InitConsumers initializes NATS consumers for ride updates
func (s *RideService) InitConsumers() error {
	// Initialize ride status consumer
	_, err := nats.NewConsumer(
		RideStatusTopic,
		"ride-service", // queue group
		s.cfg.NATS.URL,
		s.handleRideStatusUpdate,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize ride status consumer: %w", err)
	}

	// Initialize ride request consumer
	_, err = nats.NewConsumer(
		RideRequestTopic,
		"ride-service", // queue group
		s.cfg.NATS.URL,
		s.handleRideRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize ride request consumer: %w", err)
	}

	log.Println("NATS consumers initialized successfully")
	return nil
}

// handleRideStatusUpdate handles ride status update messages from NATS
func (s *RideService) handleRideStatusUpdate(msg []byte) error {
	// Implementation would parse the message and update ride status
	log.Printf("Received ride status update: %s", string(msg))
	return nil
}

// handleRideRequest handles ride request messages from NATS
func (s *RideService) handleRideRequest(msg []byte) error {
	// Implementation would parse the message and create a new ride request
	log.Printf("Received ride request: %s", string(msg))
	return nil
}
