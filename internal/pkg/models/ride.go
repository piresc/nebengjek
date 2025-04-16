package models

import (
	"time"

	"github.com/google/uuid"
)

// RideStatus represents the status of a ride
type RideStatus string

const (
	RideStatusPending   RideStatus = "pending"
	RideStatusOngoing   RideStatus = "ongoing"
	RideStatusCompleted RideStatus = "completed"
)

// Ride represents a ride record
type Ride struct {
	RideID     uuid.UUID  `json:"ride_id" db:"ride_id"`
	DriverID   uuid.UUID  `json:"driver_id" db:"driver_id"`
	CustomerID uuid.UUID  `json:"customer_id" db:"customer_id"`
	Status     RideStatus `json:"status" db:"status"`
	TotalCost  int        `json:"total_cost" db:"total_cost"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// BillingLedger represents an entry in the billing ledger
type BillingLedger struct {
	EntryID   uuid.UUID `json:"entry_id" db:"entry_id"`
	RideID    uuid.UUID `json:"ride_id" db:"ride_id"`
	Distance  float64   `json:"distance" db:"distance"`
	Cost      int       `json:"cost" db:"cost"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
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
