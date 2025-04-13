package models

import (
	"time"
)

// User represents a user in the system (either driver or customer)
type User struct {
	ID         string    `json:"id" bson:"_id" db:"id"`
	MSISDN     string    `json:"msisdn" bson:"msisdn" db:"msisdn"`
	FullName   string    `json:"fullname" bson:"fullname" db:"fullname"`
	Role       string    `json:"role" bson:"role" db:"role"` // driver or customer
	CreatedAt  time.Time `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
	IsActive   bool      `json:"is_active" bson:"is_active" db:"is_active"`
	Rating     float64   `json:"rating" bson:"rating" db:"rating"`
	DriverInfo *Driver   `json:"driver_info,omitempty" bson:"driver_info,omitempty"`
}

// Driver represents additional information for users who are drivers
type Driver struct {
	VehicleType     string    `json:"vehicle_type" bson:"vehicle_type" db:"vehicle_type"`
	VehiclePlate    string    `json:"vehicle_plate" bson:"vehicle_plate" db:"vehicle_plate"`
	VehicleModel    string    `json:"vehicle_model" bson:"vehicle_model" db:"vehicle_model"`
	VehicleColor    string    `json:"vehicle_color" bson:"vehicle_color" db:"vehicle_color"`
	LicenseNumber   string    `json:"license_number" bson:"license_number" db:"license_number"`
	Verified        bool      `json:"verified" bson:"verified" db:"verified"`
	VerifiedAt      time.Time `json:"verified_at" bson:"verified_at" db:"verified_at"`
	CurrentLocation *Location `json:"current_location,omitempty" bson:"current_location,omitempty"`
	IsAvailable     bool      `json:"is_available" bson:"is_available" db:"is_available"`
}

// Location represents a geographical location with latitude and longitude
type Location struct {
	Latitude  float64   `json:"latitude" bson:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" bson:"longitude" db:"longitude"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp" db:"timestamp"`
}
