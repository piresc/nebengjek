package gateway

import (
	"context"
	"encoding/json"
	"fmt"

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
func (g *RideGW) PublishRideStarted(ctx context.Context, ride *models.Ride) error {
	fmt.Printf("Publishing ride started event: rideID=%s, driverID=%s, customerID=%s\n",
		ride.RideID, ride.DriverID, ride.CustomerID)

	data, err := json.Marshal(ride)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectRideStarted, data)
}

// PublishRideCompleted publishes a ride completed event to NATS
func (g *RideGW) PublishRideCompleted(ctx context.Context, rideComplete models.RideComplete) error {
	fmt.Printf("Publishing ride completed event: rideID=%s, driverID=%s, customerID=%s\n",
		rideComplete.Ride.RideID, rideComplete.Ride.DriverID, rideComplete.Ride.CustomerID)

	data, err := json.Marshal(rideComplete)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectRideCompleted, data)
}
