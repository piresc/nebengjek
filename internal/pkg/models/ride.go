package models

import (
	"time"

	"github.com/google/uuid"
)

// RideStatus represents the status of a ride
type RideStatus string

const (
	RideStatusPending      RideStatus = "PENDING"
	RideStatusDriverPickup RideStatus = "PICKUP"
	RideStatusOngoing      RideStatus = "ONGOING"
	RideStatusCompleted    RideStatus = "COMPLETED"
)

// Ride represents a ride record
type Ride struct {
	RideID      uuid.UUID  `json:"ride_id" db:"ride_id"`
	DriverID    uuid.UUID  `json:"driver_id" db:"driver_id"`
	PassengerID uuid.UUID  `json:"passenger_id" db:"passenger_id"`
	Status      RideStatus `json:"status" db:"status"`
	TotalCost   int        `json:"total_cost" db:"total_cost"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type RideResp struct {
	RideID      string    `json:"ride_id"`
	DriverID    string    `json:"driver_id"`
	PassengerID string    `json:"passenger_id"`
	Status      string    `json:"status"`
	TotalCost   int       `json:"total_cost"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BillingLedger represents an entry in the billing ledger
type BillingLedger struct {
	EntryID   uuid.UUID `json:"entry_id" db:"entry_id"`
	RideID    uuid.UUID `json:"ride_id" db:"ride_id"`
	Distance  float64   `json:"distance" db:"distance"`
	Cost      int       `json:"cost" db:"cost"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RideCompleteEvent represents an event to complete a ride with adjustment
type RideCompleteEvent struct {
	RideID           string  `json:"ride_id"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
}

type RideArrivalReq struct {
	RideID           string  `json:"ride_id"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
}

// Payment represents a payment record
type Payment struct {
	PaymentID    uuid.UUID `json:"payment_id" db:"payment_id"`
	RideID       uuid.UUID `json:"ride_id" db:"ride_id"`
	AdjustedCost int       `json:"adjusted_cost" db:"adjusted_cost"`
	AdminFee     int       `json:"admin_fee" db:"admin_fee"`
	DriverPayout int       `json:"driver_payout" db:"driver_payout"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type RideComplete struct {
	Ride    Ride    `json:"ride"`
	Payment Payment `json:"payment"`
}

// RideStartTripEvent represents an event to start trip after driver picks up passenger
type RideStartTripEvent struct {
	RideID            string    `json:"ride_id"`
	DriverLocation    Location  `json:"driver_location"`
	PassengerLocation Location  `json:"passenger_location"`
	Timestamp         time.Time `json:"timestamp"`
}

// RideStartTripRequest represents a request to start a trip via HTTP
type RideStartRequest struct {
	RideID            string    `json:"ride_id"`
	DriverLocation    *Location `json:"driver_location"`
	PassengerLocation *Location `json:"passenger_location"`
}

// RidePickupEvent represents an event when a driver is on their way to pick up a passenger
type RidePickupEvent struct {
	RideID         string    `json:"ride_id"`
	DriverID       string    `json:"driver_id"`
	PassengerID    string    `json:"passenger_id"`
	DriverLocation Location  `json:"driver_location"`
	Timestamp      time.Time `json:"timestamp"`
}

type RideArrival struct {
	RideID           string  `json:"ride_id"`
	DriverID         string  `json:"driver_id"`
	PassengerID      string  `json:"passenger_id"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
}
