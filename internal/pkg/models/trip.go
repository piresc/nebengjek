package models

import (
	"time"
)

// TripStatus represents the current status of a trip
type TripStatus string

const (
	TripStatusRequested  TripStatus = "REQUESTED"
	TripStatusMatched    TripStatus = "MATCHED"
	TripStatusAccepted   TripStatus = "ACCEPTED"
	TripStatusRejected   TripStatus = "REJECTED"
	TripStatusCancelled  TripStatus = "CANCELLED"
	TripStatusInProgress TripStatus = "IN_PROGRESS"
	TripStatusCompleted  TripStatus = "COMPLETED"
	TripStatusProposed   TripStatus = "PROPOSED"
)

// Trip represents a ride-sharing trip
type Trip struct {
	ID              string     `json:"id" bson:"_id" db:"id"`
	PassengerMSISDN string     `json:"passenger_id" bson:"passenger_id" db:"passenger_id"`
	DriverMSISDN    string     `json:"driver_id,omitempty" bson:"driver_id,omitempty" db:"driver_id"`
	PickupLocation  Location   `json:"pickup_location" bson:"pickup_location"`
	DropoffLocation Location   `json:"dropoff_location" bson:"dropoff_location"`
	RequestedAt     time.Time  `json:"requested_at" bson:"requested_at" db:"requested_at"`
	MatchedAt       *time.Time `json:"matched_at,omitempty" bson:"matched_at,omitempty" db:"matched_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty" bson:"accepted_at,omitempty" db:"accepted_at"`
	StartedAt       *time.Time `json:"started_at,omitempty" bson:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" bson:"completed_at,omitempty" db:"completed_at"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty" bson:"cancelled_at,omitempty" db:"cancelled_at"`
	Status          TripStatus `json:"status" bson:"status" db:"status"`
	Fare            *Fare      `json:"fare,omitempty" bson:"fare,omitempty"`
	Distance        float64    `json:"distance" bson:"distance" db:"distance"` // in kilometers
	Duration        int        `json:"duration" bson:"duration" db:"duration"` // in minutes
	PassengerRating *float64   `json:"passenger_rating,omitempty" bson:"passenger_rating,omitempty" db:"passenger_rating"`
	DriverRating    *float64   `json:"driver_rating,omitempty" bson:"driver_rating,omitempty" db:"driver_rating"`
	Notes           string     `json:"notes,omitempty" bson:"notes,omitempty" db:"notes"`
}

// Fare represents the pricing information for a trip
type Fare struct {
	BaseFare      float64 `json:"base_fare" bson:"base_fare" db:"base_fare"`
	DistanceFare  float64 `json:"distance_fare" bson:"distance_fare" db:"distance_fare"`
	DurationFare  float64 `json:"duration_fare" bson:"duration_fare" db:"duration_fare"`
	SurgeFactor   float64 `json:"surge_factor" bson:"surge_factor" db:"surge_factor"`
	TotalFare     float64 `json:"total_fare" bson:"total_fare" db:"total_fare"`
	Currency      string  `json:"currency" bson:"currency" db:"currency"`
	PaymentMethod string  `json:"payment_method" bson:"payment_method" db:"payment_method"`
	PaymentStatus string  `json:"payment_status" bson:"payment_status" db:"payment_status"`
}
