package models

import (
	"time"
)

// Transaction represents a financial transaction record
type Transaction struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	PaymentID string    `json:"payment_id" db:"payment_id"`
	Amount    float64   `json:"amount" db:"amount"`
	Currency  string    `json:"currency" db:"currency"`
	Status    string    `json:"status" db:"status"`
	Type      string    `json:"type" db:"type"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
