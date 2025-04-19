package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideGW handles NATS publishing for ride events
type rideGW struct {
	nc *nats.Conn
}

// NewRideGW creates a new ride gateway
func NewRideGW(nc *nats.Conn) *rideGW {
	return &rideGW{
		nc: nc,
	}
}

// PublishRideStarted publishes a ride started event to NATS
func (g *rideGW) PublishRideStarted(ctx context.Context, ride *models.Ride) error {
	fmt.Printf("Publishing ride started event: rideID=%s, driverID=%s, customerID=%s\n",
		ride.RideID, ride.DriverID, ride.CustomerID)

	data, err := json.Marshal(ride)
	if err != nil {
		return err
	}
	return g.nc.Publish(constants.SubjectRideStarted, data)
}

// PublishRideCompleted publishes a ride completed event to NATS
func (g *rideGW) PublishRideCompleted(ctx context.Context, rideComplete models.RideComplete) error {
	fmt.Printf("Publishing ride completed event: rideID=%s, driverID=%s, customerID=%s\n",
		rideComplete.Ride.RideID, rideComplete.Ride.DriverID, rideComplete.Ride.CustomerID)

	data, err := json.Marshal(rideComplete)
	if err != nil {
		return err
	}
	return g.nc.Publish(constants.SubjectRideCompleted, data)
}
