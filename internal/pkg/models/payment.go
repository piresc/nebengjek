package models

import (
	"time"
)

// Payment represents a payment record
type Payment struct {
	ID            string     `json:"id" db:"id"`
	UserID        string     `json:"user_id" db:"user_id"`
	Amount        float64    `json:"amount" db:"amount"`
	Currency      string     `json:"currency" db:"currency"`
	Status        string     `json:"status" db:"status"`
	StartLocation string     `json:"start_location" db:"start_location"`
	EndLocation   string     `json:"end_location" db:"end_location"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// PaymentEvent represents a payment event for publishing
type PaymentEvent struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
