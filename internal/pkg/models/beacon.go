package models

import "time"

// BeaconRequest represents a request to toggle user beacon (availability)
type BeaconRequest struct {
	MSISDN    string  `json:"msisdn"`
	IsActive  bool    `json:"is_active"`
	Role      string  `json:"role"` // "driver" or "passenger"
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// BeaconResponse represents a response to a beacon toggle request
type BeaconResponse struct {
	Message string `json:"message"`
}

// BeaconEvent represents a beacon status change event for NATS
type BeaconEvent struct {
	MSISDN    string      `json:"msisdn"`
	IsActive  bool        `json:"is_active"`
	Role      string      `json:"role"`
	Location  GeoLocation `json:"location"`
	Timestamp time.Time   `json:"timestamp"`
}

// GeoLocation represents a geographic location
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
