package models

import (
	"time"

	"github.com/google/uuid"
)

// MatchStatus represents the status of a match
type MatchStatus string

const (
	MatchStatusPending  MatchStatus = "PENDING"
	MatchStatusAccepted MatchStatus = "ACCEPTED"
	MatchStatusRejected MatchStatus = "REJECTED"
)

// Match represents a ride-sharing match between a driver and a passenger
type Match struct {
	ID                uuid.UUID   `json:"match_id" db:"id"`
	DriverID          uuid.UUID   `json:"driver_id" db:"driver_id"`
	PassengerID       uuid.UUID   `json:"passenger_id" db:"passenger_id"`
	DriverLocation    Location    `json:"driver_location" db:"driver_location"`
	PassengerLocation Location    `json:"passenger_location" db:"passenger_location"`
	Status            MatchStatus `json:"status" db:"status"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}

// MatchDTO is used for database operations to flatten the nested Location structs
type MatchDTO struct {
	ID                 uuid.UUID   `db:"id"`
	DriverID           uuid.UUID   `db:"driver_id"`
	PassengerID        uuid.UUID   `db:"passenger_id"`
	DriverLongitude    float64     `db:"driver_longitude"`
	DriverLatitude     float64     `db:"driver_latitude"`
	PassengerLongitude float64     `db:"passenger_longitude"`
	PassengerLatitude  float64     `db:"passenger_latitude"`
	Status             MatchStatus `db:"status"`
	CreatedAt          time.Time   `db:"created_at"`
	UpdatedAt          time.Time   `db:"updated_at"`
}

// ToDTO converts a Match to a MatchDTO
func (m *Match) ToDTO() *MatchDTO {
	return &MatchDTO{
		ID:                 m.ID,
		DriverID:           m.DriverID,
		PassengerID:        m.PassengerID,
		DriverLongitude:    m.DriverLocation.Longitude,
		DriverLatitude:     m.DriverLocation.Latitude,
		PassengerLongitude: m.PassengerLocation.Longitude,
		PassengerLatitude:  m.PassengerLocation.Latitude,
		Status:             m.Status,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

// FromDTO converts a MatchDTO to a Match
func (dto *MatchDTO) ToMatch() *Match {
	return &Match{
		ID:          dto.ID,
		DriverID:    dto.DriverID,
		PassengerID: dto.PassengerID,
		DriverLocation: Location{
			Latitude:  dto.DriverLatitude,
			Longitude: dto.DriverLongitude,
			Timestamp: dto.CreatedAt,
		},
		PassengerLocation: Location{
			Latitude:  dto.PassengerLatitude,
			Longitude: dto.PassengerLongitude,
			Timestamp: dto.CreatedAt,
		},
		Status:    dto.Status,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
	}
}

type MatchProposal struct {
	ID             string      `json:"match_id"`
	PassengerID    string      `json:"passenger_id"`
	DriverID       string      `json:"driver_id"`
	UserLocation   Location    `json:"location"`
	DriverLocation Location    `json:"driver_location"`
	MatchStatus    MatchStatus `json:"match_status"`
}

// NearbyUser represents a user with their current location and distance
type NearbyUser struct {
	ID       string   `json:"id"`
	Location Location `json:"location"`
	Distance float64  `json:"distance_km"`
}
