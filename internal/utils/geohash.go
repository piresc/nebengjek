package utils

import (
	"math"
)

// GeoPoint represents a geographical point with latitude and longitude
type GeoPoint struct {
	Latitude  float64
	Longitude float64
}

// CalculateDistance calculates the distance between two points in kilometers using the Haversine formula
func CalculateDistance(point1, point2 GeoPoint) float64 {
	// Earth's radius in kilometers
	const earthRadius = 6371.0

	// Convert latitude and longitude from degrees to radians
	lat1 := point1.Latitude * math.Pi / 180.0
	lon1 := point1.Longitude * math.Pi / 180.0
	lat2 := point2.Latitude * math.Pi / 180.0
	lon2 := point2.Longitude * math.Pi / 180.0

	// Haversine formula
	dLat := lat2 - lat1
	dLon := lon2 - lon1
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c

	return distance
}
