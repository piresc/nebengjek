package usecase

import (
	"context"
	"fmt"
	"log"

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

	log.Printf("Processing location update for ride %s: lat=%.6f, long=%.6f",
		update.RideID, update.Location.Latitude, update.Location.Longitude)

	// Get last location to calculate distance
	lastLocation, err := uc.locationRepo.GetLastLocation(ctx, update.RideID)
	if err != nil {
		// If no previous location found, store this as first location
		log.Printf("No previous location found for ride %s, storing initial location", update.RideID)
		err = uc.locationRepo.StoreLocation(ctx, update.RideID, update.Location)
		if err != nil {
			return fmt.Errorf("failed to store initial location: %w", err)
		}
		return nil
	}

	log.Printf("Found previous location for ride %s: lat=%.6f, long=%.6f",
		update.RideID, lastLocation.Latitude, lastLocation.Longitude)

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

	log.Printf("Calculated distance for ride %s: %.2f km", update.RideID, distance)

	// Store new location
	err = uc.locationRepo.StoreLocation(ctx, update.RideID, update.Location)
	if err != nil {
		return fmt.Errorf("failed to store location: %w", err)
	}

	// Publish location aggregate
	log.Printf("Publishing location aggregate: ride_id=%s, distance=%.2f km", update.RideID, distance)
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
