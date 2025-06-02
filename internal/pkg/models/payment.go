package models

import (
	"time"
)

// PaymentRequest represents a request to process payment for a completed ride
type PaymentRequest struct {
	RideID           string  `json:"ride_id"`
	PassengerID      string  `json:"passenger_id"`
	TotalCost        int     `json:"total_cost"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
	AdminFeePercent  float64 `json:"admin_fee_percent"`
}

// PaymentResponse represents the response to a payment request
type PaymentResponse struct {
	Payment  Payment   `json:"payment"`
	RideID   string    `json:"ride_id"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	IssuedAt time.Time `json:"issued_at"`
}
