package usecase

import (
	"context"
	"fmt"

	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/utils"
	locationsvc "github.com/piresc/nebengjek/services/location"
)

type locationUC struct {
	locationRepo locationsvc.LocationRepo
	locationGW   locationsvc.LocationGW
}

// NewLocationUC creates a new location use case instance
func NewLocationUC(
	locationRepo locationsvc.LocationRepo,
	locationGW locationsvc.LocationGW,
) locationsvc.LocationUC {
	return &locationUC{
		locationRepo: locationRepo,
		locationGW:   locationGW,
	}
}

// StoreLocation stores a location update and publishes aggregated data
func (uc *locationUC) StoreLocation(ctx context.Context, update locationmodels.LocationUpdate) error {

	// Processing location update for ride

	// Get last location to calculate distance
	lastLocation, err := uc.locationRepo.GetLastLocation(ctx, update.RideID)
	if err != nil {
		// If no previous location found, store this as first location
		// No previous location found for ride, storing initial location
		err = uc.locationRepo.StoreLocation(ctx, update.RideID, update.Location)
		if err != nil {
			return fmt.Errorf("failed to store initial location: %w", err)
		}
		return nil
	}

	// Calculate distance using Haversine formula
	lastPoint := utils.GeoPoint{
		Latitude:  lastLocation.Latitude,
		Longitude: lastLocation.Longitude,
	}
	currentPoint := utils.GeoPoint{
		Latitude:  update.Location.Latitude,
		Longitude: update.Location.Longitude,
	}
	distance := utils.CalculateDistance(lastPoint, currentPoint)

	// Store new location
	err = uc.locationRepo.StoreLocation(ctx, update.RideID, update.Location)
	if err != nil {
		return fmt.Errorf("failed to store location: %w", err)
	}

	aggregate := locationmodels.LocationAggregate{
		RideID:    update.RideID,
		Distance:  distance,
		Latitude:  update.Location.Latitude,
		Longitude: update.Location.Longitude,
	}

	err = uc.locationGW.PublishLocationAggregate(ctx, aggregate)
	if err != nil {
		return fmt.Errorf("failed to publish location aggregate: %w", err)
	}

	return nil
}

// AddAvailableDriver adds a driver to the available drivers geo set
func (uc *locationUC) AddAvailableDriver(ctx context.Context, driverID string, location *locationmodels.Location) error {
	return uc.locationRepo.AddAvailableDriver(ctx, driverID, location)
}

// RemoveAvailableDriver removes a driver from the available drivers sets
func (uc *locationUC) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	return uc.locationRepo.RemoveAvailableDriver(ctx, driverID)
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index
func (uc *locationUC) AddAvailablePassenger(ctx context.Context, passengerID string, location *locationmodels.Location) error {
	return uc.locationRepo.AddAvailablePassenger(ctx, passengerID, location)
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
func (uc *locationUC) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	return uc.locationRepo.RemoveAvailablePassenger(ctx, passengerID)
}

// FindNearbyDrivers finds available drivers within the specified radius
func (uc *locationUC) FindNearbyDrivers(ctx context.Context, location *locationmodels.Location, radiusKm float64) ([]*matchmodels.NearbyUser, error) {
	return uc.locationRepo.FindNearbyDrivers(ctx, location, radiusKm)
}

// GetDriverLocation retrieves a driver's last known location
func (uc *locationUC) GetDriverLocation(ctx context.Context, driverID string) (locationmodels.Location, error) {
	return uc.locationRepo.GetDriverLocation(ctx, driverID)
}

// GetPassengerLocation retrieves a passenger's last known location
func (uc *locationUC) GetPassengerLocation(ctx context.Context, passengerID string) (locationmodels.Location, error) {
	return uc.locationRepo.GetPassengerLocation(ctx, passengerID)
}
