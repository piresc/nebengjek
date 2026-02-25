package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
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
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		CreatedAt: timestamp,
	}

	// Previous location
	lastLocation := &location.Location{
		Latitude:  -6.174392, // Slightly different
		Longitude: 106.826153,
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
		DoAndReturn(func(_ context.Context, aggregate location.LocationAggregate) error {
			assert.Equal(t, rideID, aggregate.RideID)
			assert.Equal(t, locationUpdate.Location.Latitude, aggregate.Latitude)
			assert.Equal(t, locationUpdate.Location.Longitude, aggregate.Longitude)
			assert.Greater(t, aggregate.Distance, 0.0) // Distance should be calculated
			return nil
		})

	// Act
	err := uc.StoreLocation(context.Background(), locationUpdate)

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
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
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
	err := uc.StoreLocation(context.Background(), locationUpdate)

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
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
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
	err := uc.StoreLocation(context.Background(), locationUpdate)

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
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
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
	err := uc.StoreLocation(context.Background(), locationUpdate)

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
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		CreatedAt: timestamp,
	}

	// Previous location exists
	lastLocation := &location.Location{
		Latitude:  -6.174392,
		Longitude: 106.826153,
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
	err := uc.StoreLocation(context.Background(), locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish location aggregate")
}

func TestStoreLocation_SecondaryStoreLocationError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	timestamp := time.Now()
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		CreatedAt: timestamp,
	}

	// Previous location exists
	lastLocation := &location.Location{
		Latitude:  -6.174392,
		Longitude: 106.826153,
			}

	expectedError := errors.New("database error")

	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(lastLocation, nil)

	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, locationUpdate.Location).
		Return(expectedError)

	// Act
	err := uc.StoreLocation(context.Background(), locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store location")
}

func TestStoreLocation_ZeroDistance(t *testing.T) {
	// Arrange - create controller and mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	timestamp := time.Now()
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		CreatedAt: timestamp,
	}

	// Previous location - exactly the same coordinates
	lastLocation := &location.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
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
		DoAndReturn(func(_ context.Context, aggregate location.LocationAggregate) error {
			assert.Equal(t, rideID, aggregate.RideID)
			assert.Equal(t, locationUpdate.Location.Latitude, aggregate.Latitude)
			assert.Equal(t, locationUpdate.Location.Longitude, aggregate.Longitude)
			assert.Equal(t, 0.0, aggregate.Distance) // Distance should be zero
			return nil
		})

	// Act
	err := uc.StoreLocation(context.Background(), locationUpdate)

	// Assert
	assert.NoError(t, err)
}

func TestStoreLocation_InvalidRideID(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	locationUpdate := location.LocationUpdate{
		RideID:   "", // Empty ride ID
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		CreatedAt: time.Now(),
	}

	// Current implementation still tries to call GetLastLocation even with empty ride ID
	// We need to set this expectation to match the current implementation
	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), "").
		Return(nil, errors.New("no location data found"))

	// Current implementation will try to store the location anyway
	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), "", locationUpdate.Location).
		Return(nil)

	// Act
	err := uc.StoreLocation(context.Background(), locationUpdate)

	// Assert
	assert.NoError(t, err) // Current implementation doesn't validate ride ID
}

func TestStoreLocation_MaxDistanceCase(t *testing.T) {
	// Arrange - create controller and mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	rideID := "ride-123"
	timestamp := time.Now()
	locationUpdate := location.LocationUpdate{
		RideID:   rideID,
		DriverID: "driver-456",
		Location: location.Location{
			Latitude:  -6.175392, // Jakarta area
			Longitude: 106.827153,
					},
		CreatedAt: timestamp,
	}

	// Previous location - New York (very far but not exactly antipodal)
	lastLocation := &location.Location{
		Latitude:  40.712776, // New York
		Longitude: -74.005974,
			}

	// Set up expectations
	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(lastLocation, nil)

	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, locationUpdate.Location).
		Return(nil)

	// Mock gateway call for location aggregate - use explicit matching instead of comparing
	mockGW.EXPECT().
		PublishLocationAggregate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, aggregate location.LocationAggregate) error {
			assert.Equal(t, rideID, aggregate.RideID)
			assert.Equal(t, locationUpdate.Location.Latitude, aggregate.Latitude)
			assert.Equal(t, locationUpdate.Location.Longitude, aggregate.Longitude)
			// Jakarta to New York should be around 16,000-18,000 km
			assert.NotZero(t, aggregate.Distance)
			assert.Greater(t, aggregate.Distance, 10000.0) // Should be very large distance (>10,000 km)
			return nil
		})

	// Act
	err := uc.StoreLocation(context.Background(), locationUpdate)

	// Assert
	assert.NoError(t, err)
}
