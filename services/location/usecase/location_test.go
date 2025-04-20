package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location/mocks"
	"github.com/stretchr/testify/assert"
)

func TestStoreLocation_Success(t *testing.T) {
	// Arrange - create controller and mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	timestamp := time.Now()
	locationUpdate := models.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: timestamp,
		},
		CreatedAt: timestamp,
	}

	// Previous location
	lastLocation := &models.Location{
		Latitude:  -6.174392, // Slightly different
		Longitude: 106.826153,
		Timestamp: timestamp.Add(-1 * time.Minute),
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(lastLocation, nil)

	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, locationUpdate.Location).
		Return(nil)

	// Mock gateway call for location aggregate
	mockGW.EXPECT().
		PublishLocationAggregate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, aggregate models.LocationAggregate) error {
			assert.Equal(t, rideID, aggregate.RideID)
			assert.Equal(t, locationUpdate.Location.Latitude, aggregate.Latitude)
			assert.Equal(t, locationUpdate.Location.Longitude, aggregate.Longitude)
			assert.Greater(t, aggregate.Distance, 0.0) // Distance should be calculated
			return nil
		})

	// Act
	err := uc.StoreLocation(locationUpdate)

	// Assert
	assert.NoError(t, err)
}

func TestStoreLocation_FirstLocation(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	locationUpdate := models.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	// No previous location found
	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(nil, errors.New("no location data found"))

	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, locationUpdate.Location).
		Return(nil)

	// Act
	err := uc.StoreLocation(locationUpdate)

	// Assert
	assert.NoError(t, err)
	// Gateway should not be called for first location - no need to set expectations
}

func TestStoreLocation_GetLastLocationError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	locationUpdate := models.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	expectedError := errors.New("database error")

	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(nil, expectedError)

	// Expect StoreLocation to be called as a fallback
	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, locationUpdate.Location).
		Return(nil)

	// Act
	err := uc.StoreLocation(locationUpdate)

	// Assert
	assert.NoError(t, err)
	// Gateway should not be called on initial location
}

func TestStoreLocation_StoreLocationError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	locationUpdate := models.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	expectedError := errors.New("database error")

	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(nil, errors.New("no location data found"))

	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, locationUpdate.Location).
		Return(expectedError)

	// Act
	err := uc.StoreLocation(locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store initial location")
}

func TestStoreLocation_PublishError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	timestamp := time.Now()
	locationUpdate := models.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: timestamp,
		},
		CreatedAt: timestamp,
	}

	// Previous location exists
	lastLocation := &models.Location{
		Latitude:  -6.174392,
		Longitude: 106.826153,
		Timestamp: timestamp.Add(-1 * time.Minute),
	}

	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(lastLocation, nil)

	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, locationUpdate.Location).
		Return(nil)

	// Mock gateway call - error
	expectedError := errors.New("publish error")
	mockGW.EXPECT().
		PublishLocationAggregate(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	err := uc.StoreLocation(locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish location aggregate")
}
