package nats

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match/mocks"
	"github.com/stretchr/testify/assert"
)

// Test the MatchHandler constructor
func TestMatchHandler_Constructor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	mockNATSClient := &natspkg.Client{}

	// Act
	handler := NewMatchHandler(mockMatchUC, mockNATSClient)

	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, mockMatchUC, handler.matchUC)
	assert.Equal(t, mockNATSClient, handler.natsClient)
	assert.NotNil(t, handler.subs)
	assert.Empty(t, handler.subs)
}

// Test beacon event handler logic directly
func TestMatchHandler_handleBeaconEvent(t *testing.T) {
	tests := []struct {
		name        string
		eventData   []byte
		expectError bool
		setupMock   func(*mocks.MockMatchUC)
	}{
		{
			name: "successful beacon event processing",
			eventData: func() []byte {
				event := models.BeaconEvent{
					UserID:   uuid.New().String(),
					IsActive: true,
					Location: models.Location{
						Latitude:  -6.175392,
						Longitude: 106.827153,
						Timestamp: time.Now(),
					},
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(event)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().HandleBeaconEvent(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "invalid JSON data",
			eventData:   []byte("invalid json"),
			expectError: true,
			setupMock:   func(m *mocks.MockMatchUC) {},
		},
		{
			name: "usecase returns error",
			eventData: func() []byte {
				event := models.BeaconEvent{
					UserID:   uuid.New().String(),
					IsActive: false,
					Location: models.Location{
						Latitude:  -6.175392,
						Longitude: 106.827153,
						Timestamp: time.Now(),
					},
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(event)
				return data
			}(),
			expectError: true,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().HandleBeaconEvent(gomock.Any(), gomock.Any()).Return(errors.New("usecase error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMatchUC := mocks.NewMockMatchUC(ctrl)
			tt.setupMock(mockMatchUC)

			mockNATSClient := &natspkg.Client{}
			handler := NewMatchHandler(mockMatchUC, mockNATSClient)

			// Act
			err := handler.handleBeaconEvent(tt.eventData)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test finder event handler logic directly
func TestMatchHandler_handleFinderEvent(t *testing.T) {
	tests := []struct {
		name        string
		eventData   []byte
		expectError bool
		setupMock   func(*mocks.MockMatchUC)
	}{
		{
			name: "successful finder event processing",
			eventData: func() []byte {
				event := models.FinderEvent{
					UserID:   uuid.New().String(),
					IsActive: true,
					Location: models.Location{
						Latitude:  -6.175392,
						Longitude: 106.827153,
						Timestamp: time.Now(),
					},
					TargetLocation: models.Location{
						Latitude:  -6.185392,
						Longitude: 106.837153,
						Timestamp: time.Now(),
					},
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(event)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().HandleFinderEvent(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "invalid JSON data",
			eventData:   []byte("invalid json"),
			expectError: true,
			setupMock:   func(m *mocks.MockMatchUC) {},
		},
		{
			name: "usecase returns error",
			eventData: func() []byte {
				event := models.FinderEvent{
					UserID:   uuid.New().String(),
					IsActive: false,
					Location: models.Location{
						Latitude:  -6.175392,
						Longitude: 106.827153,
						Timestamp: time.Now(),
					},
					TargetLocation: models.Location{
						Latitude:  -6.185392,
						Longitude: 106.837153,
						Timestamp: time.Now(),
					},
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(event)
				return data
			}(),
			expectError: true,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().HandleFinderEvent(gomock.Any(), gomock.Any()).Return(errors.New("usecase error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMatchUC := mocks.NewMockMatchUC(ctrl)
			tt.setupMock(mockMatchUC)

			mockNATSClient := &natspkg.Client{}
			handler := NewMatchHandler(mockMatchUC, mockNATSClient)

			// Act
			err := handler.handleFinderEvent(tt.eventData)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test ride pickup handler logic directly
func TestMatchHandler_handleRidePickup(t *testing.T) {
	tests := []struct {
		name        string
		eventData   []byte
		expectError bool
		setupMock   func(*mocks.MockMatchUC)
	}{
		{
			name: "successful ride pickup processing",
			eventData: func() []byte {
				rideResp := models.RideResp{
					RideID:      uuid.New().String(),
					DriverID:    uuid.New().String(),
					PassengerID: uuid.New().String(),
					Status:      "active",
					TotalCost:   0,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				data, _ := json.Marshal(rideResp)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().SetActiveRide(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().RemoveDriverFromPool(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().RemovePassengerFromPool(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "invalid JSON data",
			eventData:   []byte("invalid json"),
			expectError: true,
			setupMock:   func(m *mocks.MockMatchUC) {},
		},
		{
			name: "driver removal fails but continues",
			eventData: func() []byte {
				rideResp := models.RideResp{
					RideID:      uuid.New().String(),
					DriverID:    uuid.New().String(),
					PassengerID: uuid.New().String(),
					Status:      "active",
					TotalCost:   0,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				data, _ := json.Marshal(rideResp)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().SetActiveRide(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().RemoveDriverFromPool(gomock.Any(), gomock.Any()).Return(errors.New("driver removal failed")).Times(1)
				m.EXPECT().RemovePassengerFromPool(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "passenger removal fails but continues",
			eventData: func() []byte {
				rideResp := models.RideResp{
					RideID:      uuid.New().String(),
					DriverID:    uuid.New().String(),
					PassengerID: uuid.New().String(),
					Status:      "active",
					TotalCost:   0,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				data, _ := json.Marshal(rideResp)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().SetActiveRide(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().RemoveDriverFromPool(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().RemovePassengerFromPool(gomock.Any(), gomock.Any()).Return(errors.New("passenger removal failed")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMatchUC := mocks.NewMockMatchUC(ctrl)
			tt.setupMock(mockMatchUC)

			mockNATSClient := &natspkg.Client{}
			handler := NewMatchHandler(mockMatchUC, mockNATSClient)

			// Act
			err := handler.handleRidePickup(tt.eventData)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test ride completed handler logic directly
func TestMatchHandler_handleRideCompleted(t *testing.T) {
	tests := []struct {
		name        string
		eventData   []byte
		expectError bool
		setupMock   func(*mocks.MockMatchUC)
	}{
		{
			name: "successful ride completed processing",
			eventData: func() []byte {
				driverID := uuid.New()
				passengerID := uuid.New()
				rideComplete := models.RideComplete{
					Ride: models.Ride{
						RideID:      uuid.New(),
						DriverID:    driverID,
						PassengerID: passengerID,
						Status:      models.RideStatusCompleted,
						TotalCost:   50000,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
					Payment: models.Payment{
						PaymentID:    uuid.New(),
						RideID:       uuid.New(),
						AdjustedCost: 50000,
						AdminFee:     2500,
						DriverPayout: 47500,
						Status:       models.PaymentStatusAccepted,
						CreatedAt:    time.Now(),
					},
				}
				data, _ := json.Marshal(rideComplete)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().RemoveActiveRide(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().ReleaseDriver(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().ReleasePassenger(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:        "invalid JSON data",
			eventData:   []byte("invalid json"),
			expectError: true,
			setupMock:   func(m *mocks.MockMatchUC) {},
		},
		{
			name: "driver release fails but continues",
			eventData: func() []byte {
				driverID := uuid.New()
				passengerID := uuid.New()
				rideComplete := models.RideComplete{
					Ride: models.Ride{
						RideID:      uuid.New(),
						DriverID:    driverID,
						PassengerID: passengerID,
						Status:      models.RideStatusCompleted,
						TotalCost:   50000,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
					Payment: models.Payment{
						PaymentID:    uuid.New(),
						RideID:       uuid.New(),
						AdjustedCost: 50000,
						AdminFee:     2500,
						DriverPayout: 47500,
						Status:       models.PaymentStatusAccepted,
						CreatedAt:    time.Now(),
					},
				}
				data, _ := json.Marshal(rideComplete)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().RemoveActiveRide(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().ReleaseDriver(gomock.Any(), gomock.Any()).Return(errors.New("driver release failed")).Times(1)
				m.EXPECT().ReleasePassenger(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "passenger release fails but continues",
			eventData: func() []byte {
				driverID := uuid.New()
				passengerID := uuid.New()
				rideComplete := models.RideComplete{
					Ride: models.Ride{
						RideID:      uuid.New(),
						DriverID:    driverID,
						PassengerID: passengerID,
						Status:      models.RideStatusCompleted,
						TotalCost:   50000,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
					Payment: models.Payment{
						PaymentID:    uuid.New(),
						RideID:       uuid.New(),
						AdjustedCost: 50000,
						AdminFee:     2500,
						DriverPayout: 47500,
						Status:       models.PaymentStatusAccepted,
						CreatedAt:    time.Now(),
					},
				}
				data, _ := json.Marshal(rideComplete)
				return data
			}(),
			expectError: false,
			setupMock: func(m *mocks.MockMatchUC) {
				m.EXPECT().RemoveActiveRide(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().ReleaseDriver(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().ReleasePassenger(gomock.Any(), gomock.Any()).Return(errors.New("passenger release failed")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMatchUC := mocks.NewMockMatchUC(ctrl)
			tt.setupMock(mockMatchUC)

			mockNATSClient := &natspkg.Client{}
			handler := NewMatchHandler(mockMatchUC, mockNATSClient)

			// Act
			err := handler.handleRideCompleted(tt.eventData)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
