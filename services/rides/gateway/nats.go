package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides"
)

// RideGW handles NATS publishing for ride events
type RideGW struct {
	natsClient *natspkg.Client
}

// NewRideGW creates a new ride gateway
func NewRideGW(client *natspkg.Client) rides.RideGW {
	return &RideGW{
		natsClient: client,
	}
}

// PublishRidePickup publishes a ride pickup event to JetStream with delivery guarantees
func (g *RideGW) PublishRidePickup(ctx context.Context, ride *models.Ride) error {
	logger.InfoCtx(ctx, "Preparing to publish ride pickup event to JetStream",
		logger.String("ride_id", ride.RideID.String()),
		logger.String("driver_id", ride.DriverID.String()),
		logger.String("passenger_id", ride.PassengerID.String()),
		logger.String("status", string(ride.Status)))

	rideResponse := models.RideResp{
		RideID:      ride.RideID.String(),
		DriverID:    ride.DriverID.String(),
		PassengerID: ride.PassengerID.String(),
		Status:      string(ride.Status),
		TotalCost:   ride.TotalCost,
		CreatedAt:   ride.CreatedAt,
		UpdatedAt:   ride.UpdatedAt,
	}

	data, err := json.Marshal(rideResponse)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to marshal ride pickup response",
			logger.String("ride_id", ride.RideID.String()),
			logger.ErrorField(err))
		return fmt.Errorf("failed to marshal ride pickup response: %w", err)
	}

	logger.InfoCtx(ctx, "Marshaled ride pickup event, publishing to JetStream",
		logger.String("subject", constants.SubjectRidePickup),
		logger.String("message_size", fmt.Sprintf("%d bytes", len(data))))

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectRidePickup,
		Data:    data,
		MsgID:   fmt.Sprintf("ride-pickup-%s-%d", ride.RideID.String(), time.Now().UnixNano()),
		Timeout: 15 * time.Second, // Longer timeout for critical ride events
	}

	logger.InfoCtx(ctx, "Publishing ride pickup event to JetStream with options",
		logger.String("subject", opts.Subject),
		logger.String("msg_id", opts.MsgID),
		logger.String("timeout", opts.Timeout.String()))

	if err := g.natsClient.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish ride pickup event to JetStream",
			logger.String("ride_id", ride.RideID.String()),
			logger.String("driver_id", ride.DriverID.String()),
			logger.String("passenger_id", ride.PassengerID.String()),
			logger.String("subject", opts.Subject),
			logger.String("msg_id", opts.MsgID),
			logger.Err(err))
		return fmt.Errorf("failed to publish ride pickup event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published ride pickup event to JetStream",
		logger.String("ride_id", ride.RideID.String()),
		logger.String("driver_id", ride.DriverID.String()),
		logger.String("passenger_id", ride.PassengerID.String()),
		logger.String("status", string(ride.Status)),
		logger.String("subject", opts.Subject),
		logger.String("msg_id", opts.MsgID))

	return nil
}

// PublishRideStarted publishes a ride started event to JetStream with delivery guarantees
func (g *RideGW) PublishRideStarted(ctx context.Context, ride *models.Ride) error {
	rideResponse := models.RideResp{
		RideID:      ride.RideID.String(),
		DriverID:    ride.DriverID.String(),
		PassengerID: ride.PassengerID.String(),
		Status:      string(ride.Status),
		TotalCost:   ride.TotalCost,
		CreatedAt:   ride.CreatedAt,
		UpdatedAt:   ride.UpdatedAt,
	}

	data, err := json.Marshal(rideResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal ride started response: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectRideStarted,
		Data:    data,
		MsgID:   fmt.Sprintf("ride-started-%s-%d", ride.RideID.String(), time.Now().UnixNano()),
		Timeout: 15 * time.Second, // Longer timeout for critical ride events
	}

	if err := g.natsClient.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish ride started event to JetStream",
			logger.String("ride_id", ride.RideID.String()),
			logger.String("driver_id", ride.DriverID.String()),
			logger.String("passenger_id", ride.PassengerID.String()),
			logger.Err(err))
		return fmt.Errorf("failed to publish ride started event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published ride started event to JetStream",
		logger.String("ride_id", ride.RideID.String()),
		logger.String("driver_id", ride.DriverID.String()),
		logger.String("passenger_id", ride.PassengerID.String()),
		logger.String("status", string(ride.Status)))

	return nil
}

// PublishRideCompleted publishes a ride completed event to JetStream with delivery guarantees
func (g *RideGW) PublishRideCompleted(ctx context.Context, rideComplete models.RideComplete) error {
	data, err := json.Marshal(rideComplete)
	if err != nil {
		return fmt.Errorf("failed to marshal ride complete event: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectRideCompleted,
		Data:    data,
		MsgID:   fmt.Sprintf("ride-completed-%s-%d", rideComplete.Ride.RideID.String(), time.Now().UnixNano()),
		Timeout: 15 * time.Second, // Longer timeout for critical ride events
	}

	if err := g.natsClient.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish ride completed event to JetStream",
			logger.String("ride_id", rideComplete.Ride.RideID.String()),
			logger.String("driver_id", rideComplete.Ride.DriverID.String()),
			logger.String("passenger_id", rideComplete.Ride.PassengerID.String()),
			logger.Err(err))
		return fmt.Errorf("failed to publish ride completed event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published ride completed event to JetStream",
		logger.String("ride_id", rideComplete.Ride.RideID.String()),
		logger.String("driver_id", rideComplete.Ride.DriverID.String()),
		logger.String("passenger_id", rideComplete.Ride.PassengerID.String()),
		logger.Int("total_cost", rideComplete.Ride.TotalCost),
		logger.Int("adjusted_cost", rideComplete.Payment.AdjustedCost))

	return nil
}
