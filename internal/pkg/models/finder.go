package models

import (
	"time"
)

// FinderRequest represents a request to toggle passenger's finder (availability)
type FinderRequest struct {
	MSISDN         string   `json:"msisdn"`
	IsActive       bool     `json:"is_active"`
	Location       Location `json:"location"`
	TargetLocation Location `json:"target_location"`
}

// FinderResponse represents a response to a finder toggle request
type FinderResponse struct {
	Message string `json:"message"`
}

// FinderEvent represents a passenger's finder status change event for NATS
type FinderEvent struct {
	UserID         string    `json:"user_id"`
	IsActive       bool      `json:"is_active"`
	Location       Location  `json:"location"`
	TargetLocation Location  `json:"target_location"`
	Timestamp      time.Time `json:"timestamp"`
}
