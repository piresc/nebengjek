package gateway

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	natsserver "github.com/nats-io/nats-server/test"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
)

var testNatsURL = "nats://127.0.0.1:8369"

func TestMain(m *testing.M) {
	opts := natsserver.DefaultTestOptions
	opts.Port = 8369
	testNatsServer := natsserver.RunServer(&opts)
	code := m.Run()
	testNatsServer.Shutdown()
	os.Exit(code)
}

// TestPublishRideStarted_Success tests successful publishing of ride started events
func TestPublishRideStarted_Success(t *testing.T) {
	// Create mock NATS client
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	defer nc.Close()

	// Create test data
	ride := &models.Ride{
		RideID:     uuid.New(),
		DriverID:   uuid.New(),
		CustomerID: uuid.New(),
		Status:     models.RideStatusOngoing,
		TotalCost:  0,
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(constants.SubjectRideStarted, func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create gateway and publish message
	rideGW := NewRideGW(nc)
	err = rideGW.PublishRideStarted(context.Background(), ride)
	require.NoError(t, err)

	// Wait for the message
	select {
	case msg := <-msgCh:
		var publishedRide models.Ride
		err = json.Unmarshal(msg.Data, &publishedRide)
		require.NoError(t, err)

		assert.Equal(t, ride.RideID, publishedRide.RideID)
		assert.Equal(t, ride.DriverID, publishedRide.DriverID)
		assert.Equal(t, ride.CustomerID, publishedRide.CustomerID)
		assert.Equal(t, ride.Status, publishedRide.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}

// TestPublishRideCompleted_Success tests successful publishing of ride completed events
func TestPublishRideCompleted_Success(t *testing.T) {
	// Create mock NATS client
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	defer nc.Close()

	// Create test data
	ride := &models.Ride{
		RideID:     uuid.New(),
		DriverID:   uuid.New(),
		CustomerID: uuid.New(),
		Status:     models.RideStatusCompleted,
	}

	payment := &models.Payment{
		PaymentID:    uuid.New(),
		RideID:       ride.RideID,
		AdjustedCost: 30000,
		AdminFee:     1500,
		DriverPayout: 28500,
	}

	rideComplete := models.RideComplete{
		Ride:    *ride,
		Payment: *payment,
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(constants.SubjectRideCompleted, func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create gateway and publish message
	rideGW := NewRideGW(nc)
	err = rideGW.PublishRideCompleted(context.Background(), rideComplete)
	require.NoError(t, err)

	// Wait for the message
	select {
	case msg := <-msgCh:
		var publishedComplete models.RideComplete
		err = json.Unmarshal(msg.Data, &publishedComplete)
		require.NoError(t, err)

		assert.Equal(t, rideComplete.Ride.RideID, publishedComplete.Ride.RideID)
		assert.Equal(t, rideComplete.Ride.DriverID, publishedComplete.Ride.DriverID)
		assert.Equal(t, rideComplete.Ride.CustomerID, publishedComplete.Ride.CustomerID)
		assert.Equal(t, rideComplete.Payment.AdjustedCost, publishedComplete.Payment.AdjustedCost)
		assert.Equal(t, rideComplete.Payment.AdminFee, publishedComplete.Payment.AdminFee)
		assert.Equal(t, rideComplete.Payment.DriverPayout, publishedComplete.Payment.DriverPayout)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}
