package service

import (
	"context"
	"errors"
	"math"
	"time"
)

const (
	BaseRatePerKm = 3000 // IDR per kilometer
	AdminFeePercentage = 0.05
)

type Location struct {
	Latitude  float64
	Longitude float64
	Timestamp time.Time
}

type Trip struct {
	ID string
	DriverID string
	CustomerID string
	StartTime time.Time
	EndTime time.Time
	Locations []Location
	BaseFare float64
	AdjustedFare float64
	AdminFee float64
	FinalFare float64
}

type BillingService struct {}

func NewBillingService() *BillingService {
	return &BillingService{}
}

func (s *BillingService) CalculateFare(ctx context.Context, trip *Trip) error {
	if len(trip.Locations) < 2 {
		return errors.New("insufficient location data")
	}

	// Calculate total distance
	totalDistance := 0.0
	for i := 1; i < len(trip.Locations); i++ {
		distance := calculateDistance(
			trip.Locations[i-1].Latitude,
			trip.Locations[i-1].Longitude,
			trip.Locations[i].Latitude,
			trip.Locations[i].Longitude,
		)
		totalDistance += distance
	}

	// Calculate base fare
	trip.BaseFare = totalDistance * BaseRatePerKm

	// Set adjusted fare (can be modified by driver later)
	trip.AdjustedFare = trip.BaseFare

	// Calculate admin fee
	trip.AdminFee = trip.AdjustedFare * AdminFeePercentage

	// Calculate final fare
	trip.FinalFare = trip.AdjustedFare - trip.AdminFee

	return nil
}

func (s *BillingService) AdjustFare(ctx context.Context, trip *Trip, adjustmentPercentage float64) error {
	if adjustmentPercentage > 1.0 || adjustmentPercentage <= 0 {
		return errors.New("invalid adjustment percentage")
	}

	// Apply driver's fare adjustment
	trip.AdjustedFare = trip.BaseFare * adjustmentPercentage

	// Recalculate admin fee and final fare
	trip.AdminFee = trip.AdjustedFare * AdminFeePercentage
	trip.FinalFare = trip.AdjustedFare - trip.AdminFee

	return nil
}

// calculateDistance uses the Haversine formula to calculate distance between two points
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // Earth's radius in kilometers

	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)
	deltaLat := toRadians(lat2 - lat1)
	deltaLon := toRadians(lon2 - lon1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}