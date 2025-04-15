package models

import (
	"time"
)

// User represents a user in the system (either driver or customer)
type User struct {
	ID         string    `json:"id" bson:"_id" db:"id"`
	MSISDN     string    `json:"msisdn" bson:"msisdn" db:"msisdn"`
	FullName   string    `json:"fullname" bson:"fullname" db:"fullname"`
	Role       string    `json:"role" bson:"role" db:"role"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
	IsActive   bool      `json:"is_active" bson:"is_active" db:"is_active"`
	DriverInfo *Driver   `json:"driver_info,omitempty" bson:"driver_info,omitempty"`
}

// Driver represents additional information for users who are drivers
type Driver struct {
	UserID       string `json:"user_id" bson:"user_id" db:"user_id"`
	VehicleType  string `json:"vehicle_type" bson:"vehicle_type" db:"vehicle_type"`
	VehiclePlate string `json:"vehicle_plate" bson:"vehicle_plate" db:"vehicle_plate"`
}

// Location represents a geographical location with latitude and longitude
type Location struct {
	Latitude  float64   `json:"latitude" bson:"latitude" db:"latitude"`
	Longitude float64   `json:"longitude" bson:"longitude" db:"longitude"`
	Address   string    `json:"address,omitempty" bson:"address,omitempty" db:"address"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp" db:"timestamp"`
}
