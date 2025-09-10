package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUser_DefaultValues(t *testing.T) {
	user := &User{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, user.ID)
	assert.Equal(t, "", user.MSISDN)
	assert.Equal(t, "", user.FullName)
	assert.Equal(t, "", user.Role)
	assert.Equal(t, time.Time{}, user.CreatedAt)
	assert.Equal(t, time.Time{}, user.UpdatedAt)
	assert.Equal(t, false, user.IsActive)
	assert.Nil(t, user.DriverInfo)
	assert.Equal(t, 0.0, user.Rating)
}

func TestUser_WithValues(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	now := time.Now()

	driverInfo := &Driver{
		UserID:       driverID,
		VehicleType:  "Car",
		VehiclePlate: "B1234XYZ",
	}

	user := &User{
		ID:         userID,
		MSISDN:     "+6281234567890",
		FullName:   "John Doe",
		Role:       "driver",
		CreatedAt:  now,
		UpdatedAt:  now,
		IsActive:   true,
		DriverInfo: driverInfo,
		Rating:     4.5,
	}

	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "+6281234567890", user.MSISDN)
	assert.Equal(t, "John Doe", user.FullName)
	assert.Equal(t, "driver", user.Role)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
	assert.True(t, user.IsActive)
	assert.NotNil(t, user.DriverInfo)
	assert.Equal(t, 4.5, user.Rating)
}

func TestUser_WithPassengerRole(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	user := &User{
		ID:        userID,
		MSISDN:    "+6281234567890",
		FullName:  "Jane Smith",
		Role:      "passenger",
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  true,
	}

	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "+6281234567890", user.MSISDN)
	assert.Equal(t, "Jane Smith", user.FullName)
	assert.Equal(t, "passenger", user.Role)
	assert.True(t, user.IsActive)
	assert.Nil(t, user.DriverInfo)
	assert.Equal(t, 0.0, user.Rating)
}

func TestUser_WithInactiveStatus(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	user := &User{
		ID:        userID,
		MSISDN:    "+6281234567890",
		FullName:  "Inactive User",
		Role:      "driver",
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  false,
	}

	assert.Equal(t, userID, user.ID)
	assert.False(t, user.IsActive)
}

func TestDriver_DefaultValues(t *testing.T) {
	driver := &Driver{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, driver.UserID)
	assert.Equal(t, "", driver.VehicleType)
	assert.Equal(t, "", driver.VehiclePlate)
}

func TestDriver_WithValues(t *testing.T) {
	userID := uuid.New()

	driver := &Driver{
		UserID:       userID,
		VehicleType:  "Motorcycle",
		VehiclePlate: "B5678ABC",
	}

	assert.Equal(t, userID, driver.UserID)
	assert.Equal(t, "Motorcycle", driver.VehicleType)
	assert.Equal(t, "B5678ABC", driver.VehiclePlate)
}

func TestDriver_VehicleTypes(t *testing.T) {
	tests := []struct {
		name         string
		vehicleType  string
		vehiclePlate string
	}{
		{"Car", "Car", "B1234XYZ"},
		{"Motorcycle", "Motorcycle", "B5678ABC"},
		{"Truck", "Truck", "D9012EFG"},
		{"Van", "Van", "B3456HIJ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			driver := &Driver{
				UserID:       userID,
				VehicleType:  tt.vehicleType,
				VehiclePlate: tt.vehiclePlate,
			}

			assert.Equal(t, tt.vehicleType, driver.VehicleType)
			assert.Equal(t, tt.vehiclePlate, driver.VehiclePlate)
		})
	}
}

func TestLocation_DefaultValues(t *testing.T) {
	location := &Location{}

	// Test zero values
	assert.Equal(t, 0.0, location.Latitude)
	assert.Equal(t, 0.0, location.Longitude)
	assert.Equal(t, time.Time{}, location.Timestamp)
}

func TestLocation_WithValues(t *testing.T) {
	now := time.Now()

	location := &Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: now,
	}

	assert.Equal(t, -6.2088, location.Latitude)
	assert.Equal(t, 106.8456, location.Longitude)
	assert.Equal(t, now, location.Timestamp)
}

func TestLocation_JakartaCoordinates(t *testing.T) {
	now := time.Now()

	location := &Location{
		Latitude:  -6.1751,  // Jakarta latitude
		Longitude: 106.8650, // Jakarta longitude
		Timestamp: now,
	}

	assert.Equal(t, -6.1751, location.Latitude)
	assert.Equal(t, 106.8650, location.Longitude)
	assert.True(t, location.Latitude < 0)  // Southern hemisphere
	assert.True(t, location.Longitude > 0) // Eastern hemisphere
}

func TestLocation_BandungCoordinates(t *testing.T) {
	now := time.Now()

	location := &Location{
		Latitude:  -6.9175,  // Bandung latitude
		Longitude: 107.6191, // Bandung longitude
		Timestamp: now,
	}

	assert.Equal(t, -6.9175, location.Latitude)
	assert.Equal(t, 107.6191, location.Longitude)
	assert.True(t, location.Latitude < 0)  // Southern hemisphere
	assert.True(t, location.Longitude > 0) // Eastern hemisphere
}

func TestUser_DriverInfoRelationship(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	now := time.Now()

	driverInfo := &Driver{
		UserID:       driverID,
		VehicleType:  "Car",
		VehiclePlate: "B1234XYZ",
	}

	user := &User{
		ID:         userID,
		MSISDN:     "+6281234567890",
		FullName:   "John Doe",
		Role:       "driver",
		CreatedAt:  now,
		UpdatedAt:  now,
		IsActive:   true,
		DriverInfo: driverInfo,
		Rating:     4.5,
	}

	// Test the relationship
	assert.NotNil(t, user.DriverInfo)
	assert.Equal(t, driverID, user.DriverInfo.UserID)
	assert.Equal(t, "Car", user.DriverInfo.VehicleType)
	assert.Equal(t, "B1234XYZ", user.DriverInfo.VehiclePlate)
}

func TestUser_RatingScenarios(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	tests := []struct {
		name   string
		rating float64
		valid  bool
	}{
		{"Perfect rating", 5.0, true},
		{"Good rating", 4.5, true},
		{"Average rating", 3.0, true},
		{"Poor rating", 2.0, true},
		{"Minimum rating", 1.0, true},
		{"Zero rating", 0.0, true},
		{"Above maximum", 5.5, false},
		{"Below minimum", -0.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				ID:        userID,
				MSISDN:    "+6281234567890",
				FullName:  "Test User",
				Role:      "driver",
				CreatedAt: now,
				UpdatedAt: now,
				IsActive:  true,
				Rating:    tt.rating,
			}

			assert.Equal(t, tt.rating, user.Rating)

			if tt.valid {
				assert.True(t, user.Rating >= 0.0 && user.Rating <= 5.0)
			}
		})
	}
}

func TestUser_RoleValidation(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"Driver role", "driver", true},
		{"Passenger role", "passenger", true},
		{"Admin role", "admin", false},
		{"Empty role", "", false},
		{"Mixed case", "Driver", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				ID:        userID,
				MSISDN:    "+6281234567890",
				FullName:  "Test User",
				Role:      tt.role,
				CreatedAt: now,
				UpdatedAt: now,
				IsActive:  true,
			}

			assert.Equal(t, tt.role, user.Role)

			// Basic validation for known roles
			isValidRole := tt.role == "driver" || tt.role == "passenger"
			assert.Equal(t, tt.expected, isValidRole)
		})
	}
}

func TestUser_MSISDNFormats(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	tests := []struct {
		name   string
		msisdn string
		valid  bool
	}{
		{"Indonesia format", "+6281234567890", true},
		{"Indonesia format 2", "081234567890", true},
		{"International format", "+441234567890", true},
		{"Too short", "123", false},
		{"Empty", "", false},
		{"With spaces", "+62 812 3456 7890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				ID:        userID,
				MSISDN:    tt.msisdn,
				FullName:  "Test User",
				Role:      "driver",
				CreatedAt: now,
				UpdatedAt: now,
				IsActive:  true,
			}

			assert.Equal(t, tt.msisdn, user.MSISDN)

			// Basic validation
			isValid := len(tt.msisdn) >= 10 && len(tt.msisdn) <= 15
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestLocation_TimestampHandling(t *testing.T) {
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
			location := &Location{
				Latitude:  -6.2088,
				Longitude: 106.8456,
				Timestamp: tt.timestamp,
			}

			assert.Equal(t, tt.timestamp, location.Timestamp)

			if tt.isZero {
				assert.True(t, location.Timestamp.IsZero())
			} else {
				assert.False(t, location.Timestamp.IsZero())
			}
		})
	}
}

func TestUser_JSONTags(t *testing.T) {
	user := &User{}
	
	assert.NotNil(t, user)
	
	// Test that all expected fields are present
	assert.Equal(t, "id", getJSONTag(&User{}, "ID"))
	assert.Equal(t, "msisdn", getJSONTag(&User{}, "MSISDN"))
	assert.Equal(t, "fullname", getJSONTag(&User{}, "FullName"))
	assert.Equal(t, "role", getJSONTag(&User{}, "Role"))
	assert.Equal(t, "created_at", getJSONTag(&User{}, "CreatedAt"))
	assert.Equal(t, "updated_at", getJSONTag(&User{}, "UpdatedAt"))
	assert.Equal(t, "is_active", getJSONTag(&User{}, "IsActive"))
	assert.Equal(t, "driver_info,omitempty", getJSONTag(&User{}, "DriverInfo"))
	assert.Equal(t, "rating,omitempty", getJSONTag(&User{}, "Rating"))
}

func TestDriver_JSONTags(t *testing.T) {
	driver := &Driver{}
	
	assert.NotNil(t, driver)
	
	// Test that all expected fields are present
	assert.Equal(t, "user_id", getJSONTag(&Driver{}, "UserID"))
	assert.Equal(t, "vehicle_type", getJSONTag(&Driver{}, "VehicleType"))
	assert.Equal(t, "vehicle_plate", getJSONTag(&Driver{}, "VehiclePlate"))
}

func TestLocation_JSONTags(t *testing.T) {
	location := &Location{}
	
	assert.NotNil(t, location)
	
	// Test that all expected fields are present
	assert.Equal(t, "latitude", getJSONTag(&Location{}, "Latitude"))
	assert.Equal(t, "longitude", getJSONTag(&Location{}, "Longitude"))
	assert.Equal(t, "timestamp", getJSONTag(&Location{}, "Timestamp"))
}