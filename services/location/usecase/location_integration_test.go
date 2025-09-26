package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/services/location/mocks"
	"github.com/stretchr/testify/assert"
)

// Integration tests for location tracking and geospatial operations
func TestLocationUC_CompleteLocationTracking_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	// Test data
	rideID := uuid.New().String()
	driverID := uuid.New().String()
	initialLocation := locationmodels.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
			}

	// Step 1: Initial location update
	locationUpdate := locationmodels.LocationUpdate{
		RideID:    rideID,
		DriverID:  driverID,
		Location:  initialLocation,
		CreatedAt: time.Now(),
	}

	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(nil, errors.New("no previous location"))

	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, initialLocation).
		Return(nil)

	// Act - Step 1: Initial location
	err := uc.StoreLocation(context.Background(), locationUpdate)

	// Assert - Step 1
	assert.NoError(t, err)

	// Step 2: Subsequent location update with movement
	newLocation := locationmodels.Location{
		Latitude:  -6.2188, // Moved ~1.1km south
		Longitude: 106.8556, // Moved ~1.1km east
			}

	newLocationUpdate := locationmodels.LocationUpdate{
		RideID:    rideID,
		DriverID:  driverID,
		Location:  newLocation,
		CreatedAt: time.Now(),
	}

	mockRepo.EXPECT().
		GetLastLocation(gomock.Any(), rideID).
		Return(&initialLocation, nil)

	// Calculate distance moved (should be significant)
	mockRepo.EXPECT().
		StoreLocation(gomock.Any(), rideID, newLocation).
		Return(nil)

	// Expect location aggregate to be published
	mockGW.EXPECT().
		PublishLocationAggregate(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act - Step 2: Movement
	err = uc.StoreLocation(context.Background(), newLocationUpdate)

	// Assert - Step 2
	assert.NoError(t, err)
}

func TestLocationUC_AddRemoveAvailableDriver_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	driverID := uuid.New().String()
	location := &locationmodels.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
			}

	// Step 1: Add available driver
	mockRepo.EXPECT().
		AddAvailableDriver(gomock.Any(), driverID, location).
		Return(nil)

	// Act - Step 1: Add driver
	err := uc.AddAvailableDriver(context.Background(), driverID, location)

	// Assert - Step 1
	assert.NoError(t, err)

	// Step 2: Remove available driver
	mockRepo.EXPECT().
		RemoveAvailableDriver(gomock.Any(), driverID).
		Return(nil)

	// Act - Step 2: Remove driver
	err = uc.RemoveAvailableDriver(context.Background(), driverID)

	// Assert - Step 2
	assert.NoError(t, err)
}

func TestLocationUC_FindNearbyDrivers_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	// Search parameters
	location := &locationmodels.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
			}
	radius := 5.0 // 5km radius

	// Mock nearby drivers
	nearbyDrivers := []*matchmodels.NearbyUser{
		{
			ID: uuid.New().String(),
			Location: locationmodels.Location{
				Latitude:  -6.2188,
				Longitude: 106.8556,
							},
			Distance: 1.5,
		},
		{
			ID: uuid.New().String(),
			Location: locationmodels.Location{
				Latitude:  -6.2288,
				Longitude: 106.8656,
							},
			Distance: 3.2,
		},
	}

	mockRepo.EXPECT().
		FindNearbyDrivers(gomock.Any(), location, radius).
		Return(nearbyDrivers, nil)

	// Act
	result, err := uc.FindNearbyDrivers(context.Background(), location, radius)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	
	// Verify results are sorted by distance (closest first)
	assert.True(t, result[0].Distance <= result[1].Distance)
	
	// Verify all results have valid data
	for _, driver := range result {
		assert.NotEmpty(t, driver.ID)
		assert.True(t, driver.Distance > 0)
	}
}

func TestLocationUC_GetDriverLocation_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	driverID := uuid.New().String()
	expectedLocation := locationmodels.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
			}

	mockRepo.EXPECT().
		GetDriverLocation(gomock.Any(), driverID).
		Return(expectedLocation, nil)

	// Act
	location, err := uc.GetDriverLocation(context.Background(), driverID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedLocation.Latitude, location.Latitude)
	assert.Equal(t, expectedLocation.Longitude, location.Longitude)
}

func TestLocationUC_GetPassengerLocation_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockLocationRepo(ctrl)
	mockGW := mocks.NewMockLocationGW(ctrl)

	uc := NewLocationUC(mockRepo, mockGW)

	passengerID := uuid.New().String()
	expectedLocation := locationmodels.Location{
		Latitude:  -6.2188,
		Longitude: 106.8556,
			}

	mockRepo.EXPECT().
		GetPassengerLocation(gomock.Any(), passengerID).
		Return(expectedLocation, nil)

	// Act
	location, err := uc.GetPassengerLocation(context.Background(), passengerID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedLocation.Latitude, location.Latitude)
	assert.Equal(t, expectedLocation.Longitude, location.Longitude)
}