package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/location"
)

type locationUC struct {
	locationRepo location.LocationRepo
	locationGW   location.LocationGW
}

// NewLocationUC creates a new location use case instance
func NewLocationUC(
	locationRepo location.LocationRepo,
	locationGW location.LocationGW,
) location.LocationUC {
	return &locationUC{
		locationRepo: locationRepo,
		locationGW:   locationGW,
	}
}

// StoreLocation stores a location update and publishes aggregated data
func (uc *locationUC) StoreLocation(update models.LocationUpdate) error {
	ctx := context.Background()

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

	// logger.Info("Found previous location for ride",
	//	logger.String("ride_id", update.RideID),
	//	logger.Float64("prev_latitude", lastLocation.Latitude),
	//	logger.Float64("prev_longitude", lastLocation.Longitude))

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

	// logger.Info("Calculated distance for ride",
	//	logger.String("ride_id", update.RideID),
	//	logger.Float64("distance_km", distance))

	// Store new location
	err = uc.locationRepo.StoreLocation(ctx, update.RideID, update.Location)
	if err != nil {
		return fmt.Errorf("failed to store location: %w", err)
	}

	// Publish location aggregate
	// logger.Info("Publishing location aggregate",
	//	logger.String("ride_id", update.RideID),
	//	logger.Float64("distance_km", distance))
	aggregate := models.LocationAggregate{
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
