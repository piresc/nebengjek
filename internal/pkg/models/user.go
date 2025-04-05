package models

import (
	"time"
)

// User represents a user in the system (either driver or passenger)
type User struct {
	ID          string    `json:"id" bson:"_id" db:"id"`
	Email       string    `json:"email" bson:"email" db:"email"`
	PhoneNumber string    `json:"phone_number" bson:"phone_number" db:"phone_number"`
	FullName    string    `json:"full_name" bson:"full_name" db:"full_name"`
	Password    string    `json:"-" bson:"password" db:"password"`
	Role        string    `json:"role" bson:"role" db:"role"` // driver or passenger
	CreatedAt   time.Time `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
	IsActive    bool      `json:"is_active" bson:"is_active" db:"is_active"`
	Rating      float64   `json:"rating" bson:"rating" db:"rating"`
	DriverInfo  *Driver   `json:"driver_info,omitempty" bson:"driver_info,omitempty"`
}

// Driver represents additional information for users who are drivers
type Driver struct {
	VehicleType     string    `json:"vehicle_type" bson:"vehicle_type" db:"vehicle_type"`
	VehiclePlate    string    `json:"vehicle_plate" bson:"vehicle_plate" db:"vehicle_plate"`
	VehicleModel    string    `json:"vehicle_model" bson:"vehicle_model" db:"vehicle_model"`
	VehicleColor    string    `json:"vehicle_color" bson:"vehicle_color" db:"vehicle_color"`
	LicenseNumber   string    `json:"license_number" bson:"license_number" db:"license_number"`
	Documents       []string  `json:"documents" bson:"documents" db:"documents"`
	Verified        bool      `json:"verified" bson:"verified" db:"verified"`
	VerifiedAt      time.Time `json:"verified_at" bson:"verified_at" db:"verified_at"`
	CurrentLocation *Location `json:"current_location,omitempty" bson:"current_location,omitempty"`
	IsAvailable     bool      `json:"is_available" bson:"is_available" db:"is_available"`
}

// Location represents a geographical location with latitude and longitude
type Location struct {
	Latitude  float64   `json:"latitude" bson:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" bson:"longitude" db:"longitude"`
	Address   string    `json:"address,omitempty" bson:"address,omitempty" db:"address"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp" db:"timestamp"`
}
