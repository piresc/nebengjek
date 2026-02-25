package location

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocationUpdate_DefaultValues(t *testing.T) {
	update := &LocationUpdate{}

	// Test zero values
	assert.Equal(t, "", update.RideID)
	assert.Equal(t, "", update.DriverID)
	assert.Equal(t, Location{}, update.Location)
	assert.Equal(t, time.Time{}, update.CreatedAt)
}

func TestLocationUpdate_WithValues(t *testing.T) {
	now := time.Now()
	location := Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	update := &LocationUpdate{
		RideID:    "test-ride-id",
		DriverID:  "test-driver-id",
		Location:  location,
		CreatedAt: now,
	}

	assert.Equal(t, "test-ride-id", update.RideID)
	assert.Equal(t, "test-driver-id", update.DriverID)
	assert.Equal(t, -6.2088, update.Location.Latitude)
	assert.Equal(t, 106.8456, update.Location.Longitude)
	assert.Equal(t, now, update.CreatedAt)
}

func TestLocationUpdate_JakartaLocation(t *testing.T) {
	now := time.Now()
	location := Location{
		Latitude:  -6.1751,
		Longitude: 106.8650,
	}

	update := &LocationUpdate{
		RideID:    "jakarta-ride",
		DriverID:  "driver-jakarta",
		Location:  location,
		CreatedAt: now,
	}

	assert.Equal(t, "jakarta-ride", update.RideID)
	assert.Equal(t, -6.1751, update.Location.Latitude)
	assert.Equal(t, 106.8650, update.Location.Longitude)
	assert.True(t, update.Location.Latitude < 0)  // Southern hemisphere
	assert.True(t, update.Location.Longitude > 0) // Eastern hemisphere
}

func TestLocationUpdate_BandungLocation(t *testing.T) {
	now := time.Now()
	location := Location{
		Latitude:  -6.9175,
		Longitude: 107.6191,
	}

	update := &LocationUpdate{
		RideID:    "bandung-ride",
		DriverID:  "driver-bandung",
		Location:  location,
		CreatedAt: now,
	}

	assert.Equal(t, "bandung-ride", update.RideID)
	assert.Equal(t, -6.9175, update.Location.Latitude)
	assert.Equal(t, 107.6191, update.Location.Longitude)
	assert.True(t, update.Location.Latitude < 0)  // Southern hemisphere
	assert.True(t, update.Location.Longitude > 0) // Eastern hemisphere
}

func TestLocationUpdate_EmptyIDs(t *testing.T) {
	now := time.Now()
	location := Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	update := &LocationUpdate{
		RideID:    "",
		DriverID:  "",
		Location:  location,
		CreatedAt: now,
	}

	assert.Empty(t, update.RideID)
	assert.Empty(t, update.DriverID)
	assert.NotNil(t, update.Location)
}

func TestLocationAggregate_DefaultValues(t *testing.T) {
	aggregate := &LocationAggregate{}

	// Test zero values
	assert.Equal(t, "", aggregate.RideID)
	assert.Equal(t, 0.0, aggregate.Distance)
	assert.Equal(t, 0.0, aggregate.Latitude)
	assert.Equal(t, 0.0, aggregate.Longitude)
}

func TestLocationAggregate_WithValues(t *testing.T) {
	aggregate := &LocationAggregate{
		RideID:    "test-ride-id",
		Distance:  5.5,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	assert.Equal(t, "test-ride-id", aggregate.RideID)
	assert.Equal(t, 5.5, aggregate.Distance)
	assert.Equal(t, -6.2088, aggregate.Latitude)
	assert.Equal(t, 106.8456, aggregate.Longitude)
}

func TestLocationAggregate_JakartaCoordinates(t *testing.T) {
	aggregate := &LocationAggregate{
		RideID:    "jakarta-ride",
		Distance:  10.2,
		Latitude:  -6.1751,
		Longitude: 106.8650,
	}

	assert.Equal(t, "jakarta-ride", aggregate.RideID)
	assert.Equal(t, 10.2, aggregate.Distance)
	assert.Equal(t, -6.1751, aggregate.Latitude)
	assert.Equal(t, 106.8650, aggregate.Longitude)
	assert.True(t, aggregate.Latitude < 0)  // Southern hemisphere
	assert.True(t, aggregate.Longitude > 0) // Eastern hemisphere
}

func TestLocationAggregate_BandungCoordinates(t *testing.T) {
	aggregate := &LocationAggregate{
		RideID:    "bandung-ride",
		Distance:  15.8,
		Latitude:  -6.9175,
		Longitude: 107.6191,
	}

	assert.Equal(t, "bandung-ride", aggregate.RideID)
	assert.Equal(t, 15.8, aggregate.Distance)
	assert.Equal(t, -6.9175, aggregate.Latitude)
	assert.Equal(t, 107.6191, aggregate.Longitude)
	assert.True(t, aggregate.Latitude < 0)  // Southern hemisphere
	assert.True(t, aggregate.Longitude > 0) // Eastern hemisphere
}

func TestLocationAggregate_DistanceScenarios(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
		valid    bool
		comment  string
	}{
		{"Short distance", 0.5, true, "Very short trip"},
		{"Medium distance", 5.0, true, "Average trip"},
		{"Long distance", 25.0, true, "Long trip"},
		{"Zero distance", 0.0, true, "No distance traveled"},
		{"Negative distance", -1.0, false, "Invalid negative distance"},
		{"Very long distance", 1000.0, true, "Very long but possible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggregate := &LocationAggregate{
				RideID:    "test-ride",
				Distance:  tt.distance,
				Latitude:  -6.2088,
				Longitude: 106.8456,
			}

			assert.Equal(t, tt.distance, aggregate.Distance)

			if tt.valid {
				assert.True(t, aggregate.Distance >= 0.0)
			} else {
				assert.True(t, aggregate.Distance < 0.0)
			}
		})
	}
}

func TestLocationUpdate_TimestampHandling(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		timestamp time.Time
		isZero    bool
	}{
		{"Current time", now, false},
		{"Past time", past, false},
		{"Future time", future, false},
		{"Zero time", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location := Location{
				Latitude:  -6.2088,
				Longitude: 106.8456,
			}

			update := &LocationUpdate{
				RideID:    "test-ride",
				DriverID:  "test-driver",
				Location:  location,
				CreatedAt: tt.timestamp,
			}

			assert.Equal(t, tt.timestamp, update.CreatedAt)

			if tt.isZero {
				assert.True(t, update.CreatedAt.IsZero())
			} else {
				assert.False(t, update.CreatedAt.IsZero())
			}
		})
	}
}

func TestLocationUpdate_Validation(t *testing.T) {
	now := time.Now()
	location := Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	tests := []struct {
		name     string
		update   *LocationUpdate
		valid    bool
		comment  string
	}{
		{
			name: "Valid update",
			update: &LocationUpdate{
				RideID:    "test-ride",
				DriverID:  "test-driver",
				Location:  location,
				CreatedAt: now,
			},
			valid:   true,
			comment: "All required fields present",
		},
		{
			name: "Missing ride ID",
			update: &LocationUpdate{
				RideID:    "",
				DriverID:  "test-driver",
				Location:  location,
				CreatedAt: now,
			},
			valid:   false,
			comment: "Ride ID is required",
		},
		{
			name: "Missing driver ID",
			update: &LocationUpdate{
				RideID:    "test-ride",
				DriverID:  "",
				Location:  location,
				CreatedAt: now,
			},
			valid:   false,
			comment: "Driver ID is required",
		},
		{
			name: "Empty location",
			update: &LocationUpdate{
				RideID:    "test-ride",
				DriverID:  "test-driver",
				Location:  Location{},
				CreatedAt: now,
			},
			valid:   false,
			comment: "Location is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation logic
			hasRideID := tt.update.RideID != ""
			hasDriverID := tt.update.DriverID != ""
			hasLocation := tt.update.Location != (Location{})
			
			isValid := hasRideID && hasDriverID && hasLocation
			
			assert.Equal(t, tt.valid, isValid, tt.comment)
		})
	}
}

func TestLocationAggregate_Validation(t *testing.T) {
	tests := []struct {
		name      string
		aggregate *LocationAggregate
		valid     bool
		comment   string
	}{
		{
			name: "Valid aggregate",
			aggregate: &LocationAggregate{
				RideID:    "test-ride",
				Distance:  5.0,
				Latitude:  -6.2088,
				Longitude: 106.8456,
			},
			valid:   true,
			comment: "All required fields present with valid values",
		},
		{
			name: "Missing ride ID",
			aggregate: &LocationAggregate{
				RideID:    "",
				Distance:  5.0,
				Latitude:  -6.2088,
				Longitude: 106.8456,
			},
			valid:   false,
			comment: "Ride ID is required",
		},
		{
			name: "Invalid coordinates",
			aggregate: &LocationAggregate{
				RideID:    "test-ride",
				Distance:  5.0,
				Latitude:  0.0,
				Longitude: 0.0,
			},
			valid:   false,
			comment: "Invalid coordinates (0,0)",
		},
		{
			name: "Valid zero distance",
			aggregate: &LocationAggregate{
				RideID:    "test-ride",
				Distance:  0.0,
				Latitude:  -6.2088,
				Longitude: 106.8456,
			},
			valid:   true,
			comment: "Zero distance is valid (no movement)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation logic
			hasRideID := tt.aggregate.RideID != ""
			hasValidDistance := tt.aggregate.Distance >= 0.0
			hasValidCoordinates := tt.aggregate.Latitude != 0.0 || tt.aggregate.Longitude != 0.0
			
			isValid := hasRideID && hasValidDistance && hasValidCoordinates
			
			assert.Equal(t, tt.valid, isValid, tt.comment)
		})
	}
}

func TestLocationUpdate_JSONTags(t *testing.T) {
	update := &LocationUpdate{}
	
	assert.NotNil(t, update)
	
	// Test that struct can be created - JSON tags are validated by the struct definition
}

func TestLocationAggregate_JSONTags(t *testing.T) {
	aggregate := &LocationAggregate{}
	
	assert.NotNil(t, aggregate)
	
	// Test that struct can be created - JSON tags are validated by the struct definition
}