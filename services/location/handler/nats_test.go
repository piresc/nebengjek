package handler

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	natsserver "github.com/nats-io/nats-server/v2/test"
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

func setupNatsHandler(t *testing.T) (*LocationHandler, *mocks.MockLocationUC) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")

	locationUC := mocks.NewMockLocationUC(ctrl)
	handler := NewLocationHandler(locationUC, nc)

	return handler, locationUC
}

func TestLocationHandler_NewLocationHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	locationUC := mocks.NewMockLocationUC(ctrl)
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err)

	handler := NewLocationHandler(locationUC, nc)

	assert.NotNil(t, handler, "Handler should not be nil")
	assert.Equal(t, locationUC, handler.locationUC, "LocationUC should be properly set")
	assert.Equal(t, nc, handler.natsClient, "NATS client should be properly set")
	assert.Empty(t, handler.subs, "Subscriptions should be initialized as empty slice")
}

func TestLocationHandler_InitNATSConsumers(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	err := handler.InitNATSConsumers()
	require.NoError(t, err, "Failed to initialize NATS consumers")

	// Check if the subscription is created
	assert.NotEmpty(t, handler.subs, "Expected subscriptions to be created")
}

func TestLocationHandler_InitNATSConsumers_Error(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	// Replace the client with a closed one to force error
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err)

	nc.Close()
	handler.natsClient = nc

	err = handler.InitNATSConsumers()
	assert.Error(t, err, "Expected error when NATS connection is closed")
	assert.Contains(t, err.Error(), "failed to subscribe to location updates")
}

func TestLocationHandler_handleLocationUpdate(t *testing.T) {
	handler, locationUC := setupNatsHandler(t)

	locationUpdate := models.LocationUpdate{
		RideID: "ride123",
		Location: models.Location{
			Latitude:  1.23456,
			Longitude: 2.34567,
		},
	}

	data, err := json.Marshal(locationUpdate)
	require.NoError(t, err, "Failed to marshal location update")

	locationUC.EXPECT().StoreLocation(locationUpdate).Return(nil).Times(1)

	err = handler.handleLocationUpdate(data)
	require.NoError(t, err, "Failed to handle location update")
}

func TestLocationHandler_handleLocationUpdate_UnmarshalError(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	// Invalid JSON data
	invalidData := []byte(`{"rideID": "ride123", "location": {malformed}`)

	err := handler.handleLocationUpdate(invalidData)
	assert.Error(t, err, "Expected unmarshal error with invalid JSON")
}

func TestLocationHandler_handleLocationUpdate_StoreError(t *testing.T) {
	handler, locationUC := setupNatsHandler(t)

	locationUpdate := models.LocationUpdate{
		RideID: "ride123",
		Location: models.Location{
			Latitude:  1.23456,
			Longitude: 2.34567,
		},
	}

	data, err := json.Marshal(locationUpdate)
	require.NoError(t, err, "Failed to marshal location update")

	expectedErr := assert.AnError
	locationUC.EXPECT().StoreLocation(locationUpdate).Return(expectedErr).Times(1)

	err = handler.handleLocationUpdate(data)
	assert.Error(t, err, "Expected error when storing location fails")
	assert.Equal(t, expectedErr, err)
}
