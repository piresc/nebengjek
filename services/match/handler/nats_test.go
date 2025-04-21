package handler

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	natsserver "github.com/nats-io/nats-server/test"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match/mocks"
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

func setupNatsHandler(t *testing.T) (*MatchHandler, *mocks.MockMatchUC) {
	ctrl := gomock.NewController(t)

	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")

	matchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(matchUC, nc)

	return handler, matchUC
}

func TestMatchHandler_NewMatchHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	matchUC := mocks.NewMockMatchUC(ctrl)
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err)

	handler := NewMatchHandler(matchUC, nc)

	assert.NotNil(t, handler, "Handler should not be nil")
	assert.Equal(t, matchUC, handler.matchUC, "MatchUC should be properly set")
	assert.Equal(t, nc, handler.natsClient, "NATS client should be properly set")
	assert.Empty(t, handler.subs, "Subscriptions should be initialized as empty slice")
}

func TestMatchHandler_InitNATSConsumers(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	err := handler.InitNATSConsumers()
	require.NoError(t, err, "Failed to initialize NATS consumers")

	// Check if subscriptions are created
	assert.NotEmpty(t, handler.subs, "Expected subscriptions to be created")
}

func TestMatchHandler_InitNATSConsumers_Error(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	// Replace the client with a closed one to force error
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err)

	nc.Close()
	handler.natsClient = nc

	err = handler.InitNATSConsumers()
	assert.Error(t, err, "Expected error when NATS connection is closed")
}

func TestMatchHandler_handleBeaconEvent(t *testing.T) {
	handler, matchUC := setupNatsHandler(t)

	beaconEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "driver",
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	data, err := json.Marshal(beaconEvent)
	require.NoError(t, err, "Failed to marshal beacon event")

	matchUC.EXPECT().HandleBeaconEvent(gomock.Any()).Return(nil).Times(1)

	err = handler.handleBeaconEvent(data)
	assert.NoError(t, err, "Failed to handle beacon event")
}

func TestMatchHandler_handleBeaconEvent_UnmarshalError(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	// Invalid JSON data
	invalidData := []byte(`{"invalid json`)

	err := handler.handleBeaconEvent(invalidData)
	assert.Error(t, err, "Expected unmarshal error with invalid JSON")
}

func TestMatchHandler_handleBeaconEvent_UsecaseError(t *testing.T) {
	handler, matchUC := setupNatsHandler(t)

	beaconEvent := models.BeaconEvent{
		UserID:   uuid.New().String(),
		Role:     "driver",
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	data, err := json.Marshal(beaconEvent)
	require.NoError(t, err, "Failed to marshal beacon event")

	expectedError := errors.New("failed to process beacon event")
	matchUC.EXPECT().HandleBeaconEvent(gomock.Any()).Return(expectedError).Times(1)

	err = handler.handleBeaconEvent(data)
	assert.Error(t, err, "Expected error when handling beacon event fails")
	assert.Equal(t, expectedError, err)
}

func TestMatchHandler_handleMatchAccept(t *testing.T) {
	handler, matchUC := setupNatsHandler(t)

	matchID := uuid.New().String()
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchAccept := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
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

	data, err := json.Marshal(matchAccept)
	require.NoError(t, err, "Failed to marshal match accept")

	matchUC.EXPECT().ConfirmMatchStatus(matchID, gomock.Any()).Return(nil).Times(1)

	err = handler.handleMatchAccept(data)
	assert.NoError(t, err, "Failed to handle match accept")
}

func TestMatchHandler_handleMatchAccept_UnmarshalError(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	// Invalid JSON data
	invalidData := []byte(`{"invalid json`)

	err := handler.handleMatchAccept(invalidData)
	assert.Error(t, err, "Expected unmarshal error with invalid JSON")
}

func TestMatchHandler_handleMatchAccept_UsecaseError(t *testing.T) {
	handler, matchUC := setupNatsHandler(t)

	matchID := uuid.New().String()
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchAccept := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	data, err := json.Marshal(matchAccept)
	require.NoError(t, err, "Failed to marshal match accept")

	expectedError := errors.New("failed to confirm match")
	matchUC.EXPECT().ConfirmMatchStatus(matchID, gomock.Any()).Return(expectedError).Times(1)

	err = handler.handleMatchAccept(data)
	assert.Error(t, err, "Expected error when confirming match fails")
	assert.Equal(t, expectedError, err)
}
