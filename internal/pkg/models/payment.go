package models

import (
	"time"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusAccepted  PaymentStatus = "ACCEPTED"
	PaymentStatusRejected  PaymentStatus = "REJECTED"
	PaymentStatusProcessed PaymentStatus = "PROCESSED"
)

// PaymentRequest represents a request to process payment for a completed ride
type PaymentRequest struct {
	RideID      string `json:"ride_id"`
	PassengerID string `json:"passenger_id"`
	TotalCost   int    `json:"total_cost"`
}

// PaymentResponse represents the response to a payment request
type PaymentResponse struct {
	Payment  Payment   `json:"payment"`
	RideID   string    `json:"ride_id"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	IssuedAt time.Time `json:"issued_at"`
}

type PaymentProccessRequest struct {
	RideID    string        `json:"ride_id"`
	TotalCost int           `json:"total_cost"`
	Status    PaymentStatus `json:"status"`
}
