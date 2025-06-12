package utils

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name     string
		point1   GeoPoint
		point2   GeoPoint
		expected float64
		tolerance float64
	}{
		{
			name: "Same point",
			point1: GeoPoint{
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			point2: GeoPoint{
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			expected:  0.0,
			tolerance: 0.001,
		},
		{
			name: "Jakarta to Bandung (approximately)",
			point1: GeoPoint{
				Latitude:  -6.175392,  // Jakarta
				Longitude: 106.827153,
			},
			point2: GeoPoint{
				Latitude:  -6.914744,  // Bandung
				Longitude: 107.609810,
			},
			expected:  120.0, // Approximately 120 km
			tolerance: 10.0,   // Allow 10km tolerance
		},
		{
			name: "Short distance within Jakarta",
			point1: GeoPoint{
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			point2: GeoPoint{
				Latitude:  -6.185392,
				Longitude: 106.837153,
			},
			expected:  1.5, // Approximately 1.5 km
			tolerance: 0.5,
		},
		{
			name: "Cross equator",
			point1: GeoPoint{
				Latitude:  -1.0,
				Longitude: 100.0,
			},
			point2: GeoPoint{
				Latitude:  1.0,
				Longitude: 100.0,
			},
			expected:  222.4, // Approximately 222.4 km (2 degrees latitude)
			tolerance: 5.0,
		},
		{
			name: "Cross 180th meridian",
			point1: GeoPoint{
				Latitude:  0.0,
				Longitude: 179.0,
			},
			point2: GeoPoint{
				Latitude:  0.0,
				Longitude: -179.0,
			},
			expected:  222.4, // Approximately 222.4 km (2 degrees longitude at equator)
			tolerance: 5.0,
		},
		{
			name: "Antipodal points (maximum distance)",
			point1: GeoPoint{
				Latitude:  0.0,
				Longitude: 0.0,
			},
			point2: GeoPoint{
				Latitude:  0.0,
				Longitude: 180.0,
			},
			expected:  20015.0, // Half of Earth's circumference
			tolerance: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDistance(tt.point1, tt.point2)
			
			// Check that result is non-negative
			assert.GreaterOrEqual(t, result, 0.0, "Distance should be non-negative")
			
			// Check that result is within expected tolerance
			assert.InDelta(t, tt.expected, result, tt.tolerance, 
				"Distance should be within tolerance of expected value")
		})
	}
}

func TestCalculateDistance_EdgeCases(t *testing.T) {
	t.Run("North and South Poles", func(t *testing.T) {
		northPole := GeoPoint{Latitude: 90.0, Longitude: 0.0}
		southPole := GeoPoint{Latitude: -90.0, Longitude: 0.0}
		
		distance := CalculateDistance(northPole, southPole)
		
		// Distance between poles should be approximately half Earth's circumference
		expected := math.Pi * 6371.0 // π * Earth's radius
		assert.InDelta(t, expected, distance, 10.0, "Distance between poles should be approximately π * R")
	})

	t.Run("Same latitude, different longitude", func(t *testing.T) {
		point1 := GeoPoint{Latitude: 45.0, Longitude: 0.0}
		point2 := GeoPoint{Latitude: 45.0, Longitude: 90.0}
		
		distance := CalculateDistance(point1, point2)
		
		// At 45° latitude, 90° longitude difference should be less than at equator
		assert.Greater(t, distance, 0.0, "Distance should be positive")
		assert.Less(t, distance, 10018.0, "Distance should be less than quarter circumference")
	})

	t.Run("Very small distance", func(t *testing.T) {
		point1 := GeoPoint{Latitude: 0.0, Longitude: 0.0}
		point2 := GeoPoint{Latitude: 0.0001, Longitude: 0.0001}
		
		distance := CalculateDistance(point1, point2)
		
		// Very small distance should be calculated accurately
		assert.Greater(t, distance, 0.0, "Distance should be positive")
		assert.Less(t, distance, 0.1, "Distance should be very small")
	})
}

func TestGeoPoint_Struct(t *testing.T) {
	t.Run("GeoPoint creation", func(t *testing.T) {
		point := GeoPoint{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		}
		
		assert.Equal(t, -6.175392, point.Latitude)
		assert.Equal(t, 106.827153, point.Longitude)
	})

	t.Run("Zero values", func(t *testing.T) {
		var point GeoPoint
		
		assert.Equal(t, 0.0, point.Latitude)
		assert.Equal(t, 0.0, point.Longitude)
	})
}

// Benchmark tests for performance
func BenchmarkCalculateDistance(b *testing.B) {
	point1 := GeoPoint{Latitude: -6.175392, Longitude: 106.827153}
	point2 := GeoPoint{Latitude: -6.914744, Longitude: 107.609810}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateDistance(point1, point2)
	}
}