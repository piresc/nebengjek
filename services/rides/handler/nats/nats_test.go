package handler

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRidesHandler tests the constructor function
func TestNewRidesHandler(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	mockClient := &natspkg.Client{}
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	// Act
	handler := NewRidesHandler(mockRidesUC, mockClient, cfg)

	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, mockRidesUC, handler.ridesUC)
	assert.Equal(t, mockClient, handler.natsClient)
	assert.Equal(t, cfg, handler.cfg)
	assert.Empty(t, handler.subs)
}

// TestRidesHandler_handleMatchAccepted_Success tests successful processing of match accepted events
func TestRidesHandler_handleMatchAccepted_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
	}

	mockRidesUC.EXPECT().CreateRide(gomock.Any(), matchProposal).Return(nil)

	// Act
	matchData, err := json.Marshal(matchProposal)
	require.NoError(t, err)

	err = handler.handleMatchAccepted(matchData)

	// Assert
	require.NoError(t, err)
}

// TestRidesHandler_handleMatchAccepted_InvalidJSON tests error handling for invalid JSON
func TestRidesHandler_handleMatchAccepted_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	// Act
	invalidJSON := []byte("{invalid json}")
	err := handler.handleMatchAccepted(invalidJSON)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

// TestRidesHandler_handleMatchAccepted_CreateRideError tests error handling when CreateRide fails
func TestRidesHandler_handleMatchAccepted_CreateRideError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("create ride failed")
	mockRidesUC.EXPECT().CreateRide(gomock.Any(), matchProposal).Return(expectedError)

	// Act
	matchData, err := json.Marshal(matchProposal)
	require.NoError(t, err)

	err = handler.handleMatchAccepted(matchData)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestRidesHandler_handleLocationAggregate_Success tests successful processing of location aggregates
func TestRidesHandler_handleLocationAggregate_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	rideID := uuid.New()
	locationAggregate := models.LocationAggregate{
		RideID:   rideID.String(),
		Distance: 2.5, // Above minimum distance
	}

	expectedCost := int(2.5 * 3000) // 7500
	expectedEntry := &models.BillingLedger{
		RideID:   rideID,
		Distance: 2.5,
		Cost:     expectedCost,
	}

	mockRidesUC.EXPECT().ProcessBillingUpdate(gomock.Any(), rideID.String(), expectedEntry).Return(nil)

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = handler.handleLocationAggregate(locationData)

	// Assert
	require.NoError(t, err)
}

// TestRidesHandler_handleLocationAggregate_BelowMinDistance tests skipping processing when distance is below minimum
func TestRidesHandler_handleLocationAggregate_BelowMinDistance(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 2.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	rideID := uuid.New()
	locationAggregate := models.LocationAggregate{
		RideID:   rideID.String(),
		Distance: 1.5, // Below minimum distance
	}

	// No expectation on ProcessBillingUpdate since it should be skipped

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = handler.handleLocationAggregate(locationData)

	// Assert
	require.NoError(t, err)
}

// TestRidesHandler_handleLocationAggregate_InvalidJSON tests error handling for invalid JSON
func TestRidesHandler_handleLocationAggregate_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	// Act
	invalidJSON := []byte("{invalid json}")
	err := handler.handleLocationAggregate(invalidJSON)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

// TestRidesHandler_handleLocationAggregate_InvalidRideID tests error handling for invalid ride ID
func TestRidesHandler_handleLocationAggregate_InvalidRideID(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	locationAggregate := models.LocationAggregate{
		RideID:   "invalid-uuid",
		Distance: 2.5,
	}

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = handler.handleLocationAggregate(locationData)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid ride ID")
}

// TestRidesHandler_handleLocationAggregate_ProcessBillingError tests error handling when ProcessBillingUpdate fails
func TestRidesHandler_handleLocationAggregate_ProcessBillingError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewRidesHandler(mockRidesUC, nil, cfg)

	rideID := uuid.New()
	locationAggregate := models.LocationAggregate{
		RideID:   rideID.String(),
		Distance: 2.5,
	}

	expectedCost := int(2.5 * 3000)
	expectedEntry := &models.BillingLedger{
		RideID:   rideID,
		Distance: 2.5,
		Cost:     expectedCost,
	}

	expectedError := errors.New("billing update failed")
	mockRidesUC.EXPECT().ProcessBillingUpdate(gomock.Any(), rideID.String(), expectedEntry).Return(expectedError)

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = handler.handleLocationAggregate(locationData)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}
