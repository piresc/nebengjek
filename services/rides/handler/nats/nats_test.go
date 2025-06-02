package handler

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides/mocks"
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

func setupNatsHandler(t *testing.T) (*RidesHandler, *mocks.MockRideUC) {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	t.Cleanup(func() { nc.Close() })

	ridesUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(ridesUC, nc)
	t.Cleanup(func() {
		for _, sub := range handler.subs {
			sub.Unsubscribe()
		}
	})

	return handler, ridesUC
}
func TestRidesHandler_NewLocationHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rideUC := mocks.NewMockRideUC(ctrl)
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err)

	handler := NewRidesHandler(rideUC, nc)

	assert.NotNil(t, handler, "Handler should not be nil")
	assert.Equal(t, rideUC, handler.ridesUC, "rideUC should be properly set")
	assert.Equal(t, nc, handler.natsClient, "NATS client should be properly set")
	assert.Empty(t, handler.subs, "Subscriptions should be initialized as empty slice")
}

func TestRidesHandler_InitNATSConsumers(t *testing.T) {
	handler, _ := setupNatsHandler(t)

	err := handler.InitNATSConsumers()
	require.NoError(t, err, "Failed to initialize NATS consumers")

	// Check if the subscription is created
	assert.NotEmpty(t, handler.subs, "Expected subscriptions to be created")
}

func TestHandleMatchAccept(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler, mockUC := setupNatsHandler(t)

	matchID := uuid.New().String()
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		DriverLocation: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		UserLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: models.MatchStatusAccepted,
	}

	// Test successful match acceptance
	t.Run("success", func(t *testing.T) {
		mockUC.EXPECT().CreateRide(gomock.Any()).DoAndReturn(
			func(match models.MatchProposal) error {
				assert.Equal(t, matchProposal.ID, match.ID)
				assert.Equal(t, matchProposal.DriverID, match.DriverID)
				assert.Equal(t, matchProposal.PassengerID, match.PassengerID)
				return nil
			},
		)

		data, err := json.Marshal(matchProposal)
		require.NoError(t, err)

		err = handler.handleMatchAccept(data)
		assert.NoError(t, err)
	})

	// Test with invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		err := handler.handleMatchAccept([]byte("invalid json"))
		assert.Error(t, err)
	})

	// Test when create ride fails
	t.Run("create ride fails", func(t *testing.T) {
		mockUC.EXPECT().CreateRide(gomock.Any()).Return(assert.AnError)

		data, err := json.Marshal(matchProposal)
		require.NoError(t, err)

		err = handler.handleMatchAccept(data)
		assert.Error(t, err)
	})
}

func TestHandleLocationAggregate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler, mockUC := setupNatsHandler(t)

	validRideID := uuid.New().String()

	tests := []struct {
		name          string
		locationAgg   models.LocationAggregate
		distance      float64
		expectProcess bool
		processFails  bool
		wantErr       bool
	}{
		{
			name: "process valid update",
			locationAgg: models.LocationAggregate{
				RideID:    validRideID,
				Distance:  2.5,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			distance:      2.5,
			expectProcess: true,
			processFails:  false,
			wantErr:       false,
		},
		{
			name: "skip small distance",
			locationAgg: models.LocationAggregate{
				RideID:    validRideID,
				Distance:  0.5, // Less than 1km
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			distance:      0.5,
			expectProcess: false,
			processFails:  false,
			wantErr:       false,
		},
		{
			name: "invalid ride ID",
			locationAgg: models.LocationAggregate{
				RideID:    "invalid-uuid",
				Distance:  2.5,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			distance:      2.5,
			expectProcess: false,
			processFails:  false,
			wantErr:       true,
		},
		{
			name: "process update fails",
			locationAgg: models.LocationAggregate{
				RideID:    validRideID,
				Distance:  2.5,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			distance:      2.5,
			expectProcess: true,
			processFails:  true,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectProcess {
				expectedCost := int(tt.distance * 3000)
				expectedEntry := &models.BillingLedger{
					RideID:   uuid.MustParse(tt.locationAgg.RideID),
					Distance: tt.distance,
					Cost:     expectedCost,
				}

				if tt.processFails {
					mockUC.EXPECT().ProcessBillingUpdate(tt.locationAgg.RideID, gomock.Any()).Return(assert.AnError)
				} else {
					mockUC.EXPECT().ProcessBillingUpdate(tt.locationAgg.RideID, gomock.Any()).DoAndReturn(
						func(rideID string, entry *models.BillingLedger) error {
							assert.Equal(t, expectedEntry.RideID, entry.RideID)
							assert.Equal(t, expectedEntry.Distance, entry.Distance)
							assert.Equal(t, expectedEntry.Cost, entry.Cost)
							return nil
						},
					)
				}
			}

			data, err := json.Marshal(tt.locationAgg)
			require.NoError(t, err)

			err = handler.handleLocationAggregate(data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	// Test with invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		err := handler.handleLocationAggregate([]byte("invalid json"))
		assert.Error(t, err)
	})
}

func TestHandleRideArrived(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler, mockUC := setupNatsHandler(t)

	rideCompleteEvent := models.RideCompleteEvent{
		RideID:           "ride-123",
		AdjustmentFactor: 1.2,
	}

	// Test successful ride completion
	t.Run("success", func(t *testing.T) {
		mockUC.EXPECT().CompleteRide(rideCompleteEvent.RideID, rideCompleteEvent.AdjustmentFactor).Return(&models.Payment{}, nil)

		data, err := json.Marshal(rideCompleteEvent)
		require.NoError(t, err)

		err = handler.handleRideArrived(data)
		assert.NoError(t, err)
	})

	// Test with invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		err := handler.handleRideArrived([]byte("invalid json"))
		assert.Error(t, err)
	})

	// Test when complete ride fails
	t.Run("complete ride fails", func(t *testing.T) {
		mockUC.EXPECT().CompleteRide(rideCompleteEvent.RideID, rideCompleteEvent.AdjustmentFactor).Return(nil, assert.AnError)

		data, err := json.Marshal(rideCompleteEvent)
		require.NoError(t, err)

		err = handler.handleRideArrived(data)
		assert.Error(t, err)
	})
}

// TestIntegrationNATSHandling tests the full pipeline from NATS message to handler
func TestIntegrationNATSHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler, mockUC := setupNatsHandler(t)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	// Test match accepted integration
	t.Run("match accepted", func(t *testing.T) {
		matchID := uuid.New().String()
		driverID := uuid.New().String()
		passengerID := uuid.New().String()

		matchProposal := models.MatchProposal{
			ID:          matchID,
			DriverID:    driverID,
			PassengerID: passengerID,
			DriverLocation: models.Location{
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			UserLocation: models.Location{
				Latitude:  -6.185392,
				Longitude: 106.837153,
			},
			MatchStatus: models.MatchStatusAccepted,
		}

		mockUC.EXPECT().CreateRide(gomock.Any()).DoAndReturn(
			func(match models.MatchProposal) error {
				assert.Equal(t, matchProposal.ID, match.ID)
				assert.Equal(t, matchProposal.DriverID, match.DriverID)
				return nil
			},
		)

		nc, err := nats.Connect(testNatsURL)
		require.NoError(t, err)
		defer nc.Close()

		data, err := json.Marshal(matchProposal)
		require.NoError(t, err)

		err = nc.Publish(constants.SubjectMatchAccepted, data)
		require.NoError(t, err)
		nc.Flush()

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)
	})

	// Test location aggregate integration
	t.Run("location aggregate", func(t *testing.T) {
		validRideID := uuid.New().String()
		locationAgg := models.LocationAggregate{
			RideID:    validRideID,
			Distance:  2.5,
			Latitude:  -6.175392,
			Longitude: 106.827153,
		}

		mockUC.EXPECT().ProcessBillingUpdate(locationAgg.RideID, gomock.Any()).Return(nil)

		nc, err := nats.Connect(testNatsURL)
		require.NoError(t, err)
		defer nc.Close()

		data, err := json.Marshal(locationAgg)
		require.NoError(t, err)

		err = nc.Publish(constants.SubjectLocationAggregate, data)
		require.NoError(t, err)
		nc.Flush()

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)
	})

	// Test ride arrived integration
	t.Run("ride arrived", func(t *testing.T) {
		rideCompleteEvent := models.RideCompleteEvent{
			RideID:           "ride-123",
			AdjustmentFactor: 1.2,
		}

		mockUC.EXPECT().CompleteRide(rideCompleteEvent.RideID, rideCompleteEvent.AdjustmentFactor).Return(&models.Payment{}, nil)

		nc, err := nats.Connect(testNatsURL)
		require.NoError(t, err)
		defer nc.Close()

		data, err := json.Marshal(rideCompleteEvent)
		require.NoError(t, err)

		err = nc.Publish(constants.SubjectRideArrived, data)
		require.NoError(t, err)
		nc.Flush()

		// Give some time for the message to be processed
		time.Sleep(100 * time.Millisecond)
	})
}
