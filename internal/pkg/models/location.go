package models

import "time"

// LocationUpdate represents a location update event
type LocationUpdate struct {
	RideID    string    `json:"ride_id"`
	DriverID  string    `json:"driver_id"`
	Location  Location  `json:"location"`
	CreatedAt time.Time `json:"created_at"`
}

// LocationAggregate represents aggregated location data for billing
type LocationAggregate struct {
	RideID    string  `json:"ride_id"`
	Distance  float64 `json:"distance"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
