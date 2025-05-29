package gateway

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	natsserver "github.com/nats-io/nats-server/test"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestPublishBeaconEvent_Success tests successful publishing of beacon events
func TestPublishBeaconEvent_Success(t *testing.T) {
	// Create NATS client
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	defer nc.Close()

	// Create test data
	beaconEvent := &models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "driver",
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(constants.SubjectUserBeacon, func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create gateway and publish message
	userGW := NewUserGW(nc, "http://localhost:8080")
	ctx := context.Background()
	err = userGW.PublishBeaconEvent(ctx, beaconEvent)
	require.NoError(t, err)

	// Wait for the message and verify contents
	select {
	case msg := <-msgCh:
		var receivedEvent models.BeaconEvent
		err = json.Unmarshal(msg.Data, &receivedEvent)
		require.NoError(t, err)

		assert.Equal(t, beaconEvent.UserID, receivedEvent.UserID)
		assert.Equal(t, beaconEvent.IsActive, receivedEvent.IsActive)
		assert.Equal(t, beaconEvent.Location.Latitude, receivedEvent.Location.Latitude)
		assert.Equal(t, beaconEvent.Location.Longitude, receivedEvent.Location.Longitude)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}

// TestMatchAccept_Success tests successful publishing of match acceptance events
func TestMatchAccept_Success(t *testing.T) {
	// Create NATS client
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	defer nc.Close()

	// Create test data
	matchProposal := &models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
		DriverLocation: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		UserLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create gateway and publish message
	userGW := NewUserGW(nc, "http://localhost:8080")
	err = userGW.MatchAccept(matchProposal)
	require.NoError(t, err)

	// Wait for the message and verify contents
	select {
	case msg := <-msgCh:
		var receivedMatch models.MatchProposal
		err = json.Unmarshal(msg.Data, &receivedMatch)
		require.NoError(t, err)

		assert.Equal(t, matchProposal.ID, receivedMatch.ID)
		assert.Equal(t, matchProposal.DriverID, receivedMatch.DriverID)
		assert.Equal(t, matchProposal.PassengerID, receivedMatch.PassengerID)
		assert.Equal(t, matchProposal.MatchStatus, receivedMatch.MatchStatus)
		assert.Equal(t, matchProposal.DriverLocation.Latitude, receivedMatch.DriverLocation.Latitude)
		assert.Equal(t, matchProposal.DriverLocation.Longitude, receivedMatch.DriverLocation.Longitude)
		assert.Equal(t, matchProposal.UserLocation.Latitude, receivedMatch.UserLocation.Latitude)
		assert.Equal(t, matchProposal.UserLocation.Longitude, receivedMatch.UserLocation.Longitude)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}

// TestPublishLocationUpdate_Success tests successful publishing of location update events
func TestPublishLocationUpdate_Success(t *testing.T) {
	// Create NATS client
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	defer nc.Close()

	// Create test data
	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: uuid.New().String(),
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(constants.SubjectLocationUpdate, func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create gateway and publish message
	userGW := NewUserGW(nc, "http://localhost:8080")
	err = userGW.PublishLocationUpdate(context.Background(), locationUpdate)
	require.NoError(t, err)

	// Wait for the message and verify contents
	select {
	case msg := <-msgCh:
		var receivedUpdate models.LocationUpdate
		err = json.Unmarshal(msg.Data, &receivedUpdate)
		require.NoError(t, err)

		assert.Equal(t, locationUpdate.RideID, receivedUpdate.RideID)
		assert.Equal(t, locationUpdate.DriverID, receivedUpdate.DriverID)
		assert.Equal(t, locationUpdate.Location.Latitude, receivedUpdate.Location.Latitude)
		assert.Equal(t, locationUpdate.Location.Longitude, receivedUpdate.Location.Longitude)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}

// TestPublishRideArrived_Success tests successful publishing of ride arrived events
func TestPublishRideArrived_Success(t *testing.T) {
	// Create NATS client
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	defer nc.Close()

	// Create test data
	rideCompleteEvent := &models.RideCompleteEvent{
		RideID:           uuid.New().String(),
		AdjustmentFactor: 1.2,
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(constants.SubjectRideArrived, func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create gateway and publish message
	userGW := NewUserGW(nc, "http://localhost:8080")
	err = userGW.PublishRideArrived(context.Background(), rideCompleteEvent)
	require.NoError(t, err)

	// Wait for the message and verify contents
	select {
	case msg := <-msgCh:
		var receivedEvent models.RideCompleteEvent
		err = json.Unmarshal(msg.Data, &receivedEvent)
		require.NoError(t, err)

		assert.Equal(t, rideCompleteEvent.RideID, receivedEvent.RideID)
		assert.Equal(t, rideCompleteEvent.AdjustmentFactor, receivedEvent.AdjustmentFactor)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}
