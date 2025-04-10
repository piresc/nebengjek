package repository

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/billing"
)

// LocationRepoImpl implements the LocationRepo interface
type LocationRepoImpl struct{}

// NewLocationRepository creates a new location repository for billing service
func NewLocationRepository() billing.LocationRepo {
	return &LocationRepoImpl{}
}

// CalculateDistance calculates the distance between two locations
func (r *LocationRepoImpl) CalculateDistance(ctx context.Context, startLocation, endLocation string) (float64, error) {
	// In a real implementation, this would likely query a database or external service
	// to get the actual coordinates for the location strings.
	// For this implementation, we'll parse the locations as comma-separated lat,lng pairs

	// Parse start location
	var startLat, startLng float64
	_, err := fmt.Sscanf(startLocation, "%f,%f", &startLat, &startLng)
	if err != nil {
		return 0, fmt.Errorf("invalid start location format: %w", err)
	}

	// Parse end location
	var endLat, endLng float64
	_, err = fmt.Sscanf(endLocation, "%f,%f", &endLat, &endLng)
	if err != nil {
		return 0, fmt.Errorf("invalid end location format: %w", err)
	}

	// Calculate distance using the geohash utility
	startPoint := utils.GeoPoint{Latitude: startLat, Longitude: startLng}
	endPoint := utils.GeoPoint{Latitude: endLat, Longitude: endLng}

	distance := utils.CalculateDistance(startPoint, endPoint)
	return distance, nil
}
