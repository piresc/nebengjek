package gateway

import (
	"context"
	"encoding/json"

	"github.com/piresc/nebengjek/internal/pkg/constants"
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

// PublishRideStarted publishes a ride started event to NATS
func (g *RideGW) PublishRidePickup(ctx context.Context, ride *models.Ride) error {

	RideResponse := models.RideResp{
		RideID:      ride.RideID.String(),
		DriverID:    ride.DriverID.String(),
		PassengerID: ride.PassengerID.String(),
		Status:      string(ride.Status),
		TotalCost:   ride.TotalCost,
		CreatedAt:   ride.CreatedAt,
		UpdatedAt:   ride.UpdatedAt,
	}
	data, err := json.Marshal(RideResponse)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectRidePickup, data)
}

// PublishRideStarted publishes a ride started event to NATS
func (g *RideGW) PublishRideStarted(ctx context.Context, ride *models.Ride) error {

	RideResponse := models.RideResp{
		RideID:      ride.RideID.String(),
		DriverID:    ride.DriverID.String(),
		PassengerID: ride.PassengerID.String(),
		Status:      string(ride.Status),
		TotalCost:   ride.TotalCost,
		CreatedAt:   ride.CreatedAt,
		UpdatedAt:   ride.UpdatedAt,
	}
	data, err := json.Marshal(RideResponse)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectRideStarted, data)
}

// PublishRideCompleted publishes a ride completed event to NATS
func (g *RideGW) PublishRideCompleted(ctx context.Context, rideComplete models.RideComplete) error {

	data, err := json.Marshal(rideComplete)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectRideCompleted, data)
}
