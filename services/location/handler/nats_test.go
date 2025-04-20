package handler

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	natsserver "github.com/nats-io/nats-server/test"
)

var (
	testNatsServer *server.Server
	testNatsURL    = "nats://127.0.0.1:8369"
)

func TestMain(m *testing.M) {
	testNatsServer = RunServerOnPort(8369)
	code := m.Run()
	testNatsServer.Shutdown()
	os.Exit(code)
}

func RunServerOnPort(port int) *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = port
	return RunServerWithOptions(&opts)
}

func RunServerWithOptions(opts *server.Options) *server.Server {
	return natsserver.RunServer(opts)
}

func setupNatsConn(t *testing.T) *nats.Conn {
	nc, err := nats.Connect(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	return nc
}

// setupTestNatsHandler creates a test NATS handler with mocked dependencies
func setupTestNatsHandler(t *testing.T, ctrl *gomock.Controller) (*NatsHandler, *mocks.MockLocationUC) {
	mockUC := mocks.NewMockLocationUC(ctrl)

	cfg := &models.Config{
		NATS: models.NATSConfig{
			URL: testNatsURL,
		},
	}

	handler, err := NewNatsHandler(mockUC, cfg)
	require.NoError(t, err, "Failed to create NATS handler")

	return handler, mockUC
}

func TestNewNatsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUC := mocks.NewMockLocationUC(ctrl)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "success",
			url:     testNatsURL,
			wantErr: false,
		},
		{
			name:    "invalid URL",
			url:     "invalid://url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &models.Config{
				NATS: models.NATSConfig{
					URL: tt.url,
				},
			}

			handler, err := NewNatsHandler(mockUC, cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, handler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, handler)
				assert.Equal(t, mockUC, handler.locationUC)
				assert.NotNil(t, handler.natsClient)
				assert.NotNil(t, handler.subs)
			}
		})
	}
}

func TestInitNATSConsumers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler, _ := setupTestNatsHandler(t, ctrl)
	defer handler.Close()

	err := handler.InitNATSConsumers()
	assert.NoError(t, err)
	assert.Len(t, handler.subs, 1, "Should have 1 subscription")
}

func TestHandleLocationUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler, mockUC := setupTestNatsHandler(t, ctrl)
	defer handler.Close()

	locationUpdate := models.LocationUpdate{
		RideID: "ride-123",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	// Test successful location update
	t.Run("success", func(t *testing.T) {
		mockUC.EXPECT().StoreLocation(gomock.Any()).DoAndReturn(
			func(update models.LocationUpdate) error {
				assert.Equal(t, locationUpdate.RideID, update.RideID)
				assert.Equal(t, locationUpdate.Location.Latitude, update.Location.Latitude)
				assert.Equal(t, locationUpdate.Location.Longitude, update.Location.Longitude)
				return nil
			},
		)

		data, err := json.Marshal(locationUpdate)
		require.NoError(t, err)

		err = handler.handleLocationUpdate(data)
		assert.NoError(t, err)
	})

	// Test with invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		err := handler.handleLocationUpdate([]byte("invalid json"))
		assert.Error(t, err)
	})

	// Test when store location fails
	t.Run("store location fails", func(t *testing.T) {
		mockUC.EXPECT().StoreLocation(gomock.Any()).Return(assert.AnError)

		data, err := json.Marshal(locationUpdate)
		require.NoError(t, err)

		err = handler.handleLocationUpdate(data)
		assert.Error(t, err)
	})
}

func TestClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler, _ := setupTestNatsHandler(t, ctrl)

	// Add a subscription to test unsubscribe
	err := handler.InitNATSConsumers()
	require.NoError(t, err)
	assert.Len(t, handler.subs, 1, "Should have 1 subscription")

	handler.Close()

	// Verify all subscriptions are closed by trying to unsubscribe again
	for _, sub := range handler.subs {
		err := sub.Unsubscribe()
		assert.Error(t, err, "Should fail to unsubscribe already unsubscribed subscription")
	}
}

// TestIntegrationHandleLocationUpdate tests the full pipeline from NATS message to handler
func TestIntegrationHandleLocationUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUC := mocks.NewMockLocationUC(ctrl)

	cfg := &models.Config{
		NATS: models.NATSConfig{
			URL: testNatsURL,
		},
	}

	handler, err := NewNatsHandler(mockUC, cfg)
	require.NoError(t, err)
	defer handler.Close()

	err = handler.InitNATSConsumers()
	require.NoError(t, err)

	locationUpdate := models.LocationUpdate{
		RideID: "ride-123",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
	}

	// Setup expectation for LocationUC.StoreLocation
	mockUC.EXPECT().StoreLocation(gomock.Any()).DoAndReturn(
		func(update models.LocationUpdate) error {
			assert.Equal(t, locationUpdate.RideID, update.RideID)
			assert.Equal(t, locationUpdate.Location.Latitude, update.Location.Latitude)
			assert.Equal(t, locationUpdate.Location.Longitude, update.Location.Longitude)
			return nil
		},
	)

	// Create a NATS client to publish a message
	nc, err := nats.Connect(testNatsURL)
	require.NoError(t, err)
	defer nc.Close()

	// Publish a location update message
	data, err := json.Marshal(locationUpdate)
	require.NoError(t, err)

	err = nc.Publish(constants.SubjectLocationUpdate, data)
	require.NoError(t, err)
	nc.Flush()

	// Give some time for the message to be processed
	time.Sleep(100 * time.Millisecond)
}
