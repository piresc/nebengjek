package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location/mocks"
	"github.com/stretchr/testify/assert"
)

// Test the LocationHandler constructor
func TestLocationHandler_Constructor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLocationUC := mocks.NewMockLocationUC(ctrl)
	mockNATSClient := &natspkg.Client{}
	mockNRApp := &newrelic.Application{}

	// Act
	handler := NewLocationHandler(mockLocationUC, mockNATSClient, mockNRApp)

	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, mockLocationUC, handler.locationUC)
	assert.Equal(t, mockNATSClient, handler.natsClient)
	assert.Equal(t, mockNRApp, handler.nrApp)
	assert.NotNil(t, handler.subs)
	assert.Empty(t, handler.subs)
}

// Test location update handler logic directly
func TestLocationHandler_handleLocationUpdate(t *testing.T) {
	tests := []struct {
		name        string
		eventData   []byte
		expectError bool
		setupMock   func(*mocks.MockLocationUC)
	}{
		{
			name: "successful location update processing",
			eventData: func() []byte {
				update := models.LocationUpdate{
					RideID:   uuid.New().String(),
					DriverID: uuid.New().String(),
					Location: models.Location{
						Latitude:  -6.175392,
						Longitude: 106.827153,
						Timestamp: time.Now(),
					},
					CreatedAt: time.Now(),
				}
				data, _ := json.Marshal(update)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockLocationUC) {
				m.EXPECT().StoreLocation(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "invalid JSON data",
			eventData:   []byte("invalid json"),
			expectError: true,
			setupMock:   func(m *mocks.MockLocationUC) {},
		},
		{
			name: "usecase returns error",
			eventData: func() []byte {
				update := models.LocationUpdate{
					RideID:   uuid.New().String(),
					DriverID: uuid.New().String(),
					Location: models.Location{
						Latitude:  -6.175392,
						Longitude: 106.827153,
						Timestamp: time.Now(),
					},
					CreatedAt: time.Now(),
				}
				data, _ := json.Marshal(update)
				return data
			}(),
			expectError: true,
			setupMock: func(m *mocks.MockLocationUC) {
				m.EXPECT().StoreLocation(gomock.Any(), gomock.Any()).Return(errors.New("usecase error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLocationUC := mocks.NewMockLocationUC(ctrl)
			tt.setupMock(mockLocationUC)

			mockNATSClient := &natspkg.Client{}
			mockNRApp := &newrelic.Application{}
			handler := NewLocationHandler(mockLocationUC, mockNATSClient, mockNRApp)

			// Act
			err := handler.handleLocationUpdate(context.Background(), tt.eventData)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
