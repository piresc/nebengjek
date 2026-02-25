package match

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
	"github.com/stretchr/testify/assert"
)

func TestMatchStatus_Constants(t *testing.T) {
	// Test that all status constants have valid values
	assert.Equal(t, "PENDING", string(MatchStatusPending))
	assert.Equal(t, "DRIVER_CONFIRMED", string(MatchStatusDriverConfirmed))
	assert.Equal(t, "PASSENGER_CONFIRMED", string(MatchStatusPassengerConfirmed))
	assert.Equal(t, "ACCEPTED", string(MatchStatusAccepted))
	assert.Equal(t, "REJECTED", string(MatchStatusRejected))
}

func TestMatchStatus_StringConversion(t *testing.T) {
	tests := []struct {
		status   MatchStatus
		expected string
	}{
		{MatchStatusPending, "PENDING"},
		{MatchStatusDriverConfirmed, "DRIVER_CONFIRMED"},
		{MatchStatusPassengerConfirmed, "PASSENGER_CONFIRMED"},
		{MatchStatusAccepted, "ACCEPTED"},
		{MatchStatusRejected, "REJECTED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestMatch_ToDTO(t *testing.T) {
	driverID := uuid.New()
	passengerID := uuid.New()
	matchID := uuid.New()
	now := time.Now()

	match := &Match{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		DriverLocation: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		PassengerLocation: location.Location{
			Latitude:  -6.1751,
			Longitude: 106.8650,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude:  106.8295,
		},
		Status:             MatchStatusPending,
		DriverConfirmed:    false,
		PassengerConfirmed: false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	dto := match.ToDTO()

	assert.NotNil(t, dto)
	assert.Equal(t, matchID, dto.ID)
	assert.Equal(t, driverID, dto.DriverID)
	assert.Equal(t, passengerID, dto.PassengerID)
	assert.Equal(t, -6.2088, dto.DriverLatitude)
	assert.Equal(t, 106.8456, dto.DriverLongitude)
	assert.Equal(t, -6.1751, dto.PassengerLatitude)
	assert.Equal(t, 106.8650, dto.PassengerLongitude)
	assert.Equal(t, -6.2297, dto.TargetLatitude)
	assert.Equal(t, 106.8295, dto.TargetLongitude)
	assert.Equal(t, MatchStatusPending, dto.Status)
	assert.Equal(t, false, dto.DriverConfirmed)
	assert.Equal(t, false, dto.PassengerConfirmed)
	assert.Equal(t, now, dto.CreatedAt)
	assert.Equal(t, now, dto.UpdatedAt)
}

func TestMatchDTO_ToMatch(t *testing.T) {
	driverID := uuid.New()
	passengerID := uuid.New()
	matchID := uuid.New()
	now := time.Now()

	dto := &MatchDTO{
		ID:                 matchID,
		DriverID:           driverID,
		PassengerID:        passengerID,
		DriverLatitude:     -6.2088,
		DriverLongitude:    106.8456,
		PassengerLatitude:  -6.1751,
		PassengerLongitude: 106.8650,
		TargetLatitude:     -6.2297,
		TargetLongitude:    106.8295,
		Status:             MatchStatusPending,
		DriverConfirmed:    false,
		PassengerConfirmed: false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	match := dto.ToMatch()

	assert.NotNil(t, match)
	assert.Equal(t, matchID, match.ID)
	assert.Equal(t, driverID, match.DriverID)
	assert.Equal(t, passengerID, match.PassengerID)
	assert.Equal(t, -6.2088, match.DriverLocation.Latitude)
	assert.Equal(t, 106.8456, match.DriverLocation.Longitude)
	assert.Equal(t, -6.1751, match.PassengerLocation.Latitude)
	assert.Equal(t, 106.8650, match.PassengerLocation.Longitude)
	assert.Equal(t, -6.2297, match.TargetLocation.Latitude)
	assert.Equal(t, 106.8295, match.TargetLocation.Longitude)
	assert.Equal(t, MatchStatusPending, match.Status)
	assert.Equal(t, false, match.DriverConfirmed)
	assert.Equal(t, false, match.PassengerConfirmed)
	assert.Equal(t, now, match.CreatedAt)
	assert.Equal(t, now, match.UpdatedAt)
}

func TestMatch_ToDTO_ToMatch_RoundTrip(t *testing.T) {
	driverID := uuid.New()
	passengerID := uuid.New()
	matchID := uuid.New()
	now := time.Now()

	originalMatch := &Match{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		DriverLocation: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		PassengerLocation: location.Location{
			Latitude:  -6.1751,
			Longitude: 106.8650,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude:  106.8295,
		},
		Status:             MatchStatusDriverConfirmed,
		DriverConfirmed:    true,
		PassengerConfirmed: false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Convert to DTO and back
	dto := originalMatch.ToDTO()
	recoveredMatch := dto.ToMatch()

	// Verify all fields are preserved
	assert.Equal(t, originalMatch.ID, recoveredMatch.ID)
	assert.Equal(t, originalMatch.DriverID, recoveredMatch.DriverID)
	assert.Equal(t, originalMatch.PassengerID, recoveredMatch.PassengerID)
	assert.Equal(t, originalMatch.DriverLocation.Latitude, recoveredMatch.DriverLocation.Latitude)
	assert.Equal(t, originalMatch.DriverLocation.Longitude, recoveredMatch.DriverLocation.Longitude)
	assert.Equal(t, originalMatch.PassengerLocation.Latitude, recoveredMatch.PassengerLocation.Latitude)
	assert.Equal(t, originalMatch.PassengerLocation.Longitude, recoveredMatch.PassengerLocation.Longitude)
	assert.Equal(t, originalMatch.TargetLocation.Latitude, recoveredMatch.TargetLocation.Latitude)
	assert.Equal(t, originalMatch.TargetLocation.Longitude, recoveredMatch.TargetLocation.Longitude)
	assert.Equal(t, originalMatch.Status, recoveredMatch.Status)
	assert.Equal(t, originalMatch.DriverConfirmed, recoveredMatch.DriverConfirmed)
	assert.Equal(t, originalMatch.PassengerConfirmed, recoveredMatch.PassengerConfirmed)
	assert.Equal(t, originalMatch.CreatedAt, recoveredMatch.CreatedAt)
	assert.Equal(t, originalMatch.UpdatedAt, recoveredMatch.UpdatedAt)
}

func TestMatch_DefaultValues(t *testing.T) {
	match := &Match{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, match.ID)
	assert.Equal(t, uuid.UUID{}, match.DriverID)
	assert.Equal(t, uuid.UUID{}, match.PassengerID)
	assert.Equal(t, location.Location{}, match.DriverLocation)
	assert.Equal(t, location.Location{}, match.PassengerLocation)
	assert.Equal(t, location.Location{}, match.TargetLocation)
	assert.Equal(t, MatchStatus(""), match.Status)
	assert.Equal(t, false, match.DriverConfirmed)
	assert.Equal(t, false, match.PassengerConfirmed)
	assert.Equal(t, time.Time{}, match.CreatedAt)
	assert.Equal(t, time.Time{}, match.UpdatedAt)
}

func TestMatchDTO_DefaultValues(t *testing.T) {
	dto := &MatchDTO{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, dto.ID)
	assert.Equal(t, uuid.UUID{}, dto.DriverID)
	assert.Equal(t, uuid.UUID{}, dto.PassengerID)
	assert.Equal(t, 0.0, dto.DriverLatitude)
	assert.Equal(t, 0.0, dto.DriverLongitude)
	assert.Equal(t, 0.0, dto.PassengerLatitude)
	assert.Equal(t, 0.0, dto.PassengerLongitude)
	assert.Equal(t, 0.0, dto.TargetLatitude)
	assert.Equal(t, 0.0, dto.TargetLongitude)
	assert.Equal(t, MatchStatus(""), dto.Status)
	assert.Equal(t, false, dto.DriverConfirmed)
	assert.Equal(t, false, dto.PassengerConfirmed)
	assert.Equal(t, time.Time{}, dto.CreatedAt)
	assert.Equal(t, time.Time{}, dto.UpdatedAt)
}

func TestMatchProposal_Structure(t *testing.T) {
	proposal := &MatchProposal{
		ID:             "test-match-id",
		PassengerID:    "test-passenger-id",
		DriverID:       "test-driver-id",
		UserLocation:   location.Location{Latitude: -6.1751, Longitude: 106.8650},
		DriverLocation: location.Location{Latitude: -6.2088, Longitude: 106.8456},
		TargetLocation: location.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    MatchStatusPending,
	}

	assert.Equal(t, "test-match-id", proposal.ID)
	assert.Equal(t, "test-passenger-id", proposal.PassengerID)
	assert.Equal(t, "test-driver-id", proposal.DriverID)
	assert.Equal(t, -6.1751, proposal.UserLocation.Latitude)
	assert.Equal(t, 106.8650, proposal.UserLocation.Longitude)
	assert.Equal(t, -6.2088, proposal.DriverLocation.Latitude)
	assert.Equal(t, 106.8456, proposal.DriverLocation.Longitude)
	assert.Equal(t, -6.2297, proposal.TargetLocation.Latitude)
	assert.Equal(t, 106.8295, proposal.TargetLocation.Longitude)
	assert.Equal(t, MatchStatusPending, proposal.MatchStatus)
}

func TestMatchConfirmRequest_Structure(t *testing.T) {
	request := &MatchConfirmRequest{
		ID:     "test-match-id",
		UserID: "test-user-id",
		Role:   "driver",
		Status: "confirmed",
	}

	assert.Equal(t, "test-match-id", request.ID)
	assert.Equal(t, "test-user-id", request.UserID)
	assert.Equal(t, "driver", request.Role)
	assert.Equal(t, "confirmed", request.Status)
}

func TestNearbyUser_Structure(t *testing.T) {
	user := &NearbyUser{
		ID:       "test-user-id",
		Location: location.Location{Latitude: -6.1751, Longitude: 106.8650},
		Distance: 2.5,
	}

	assert.Equal(t, "test-user-id", user.ID)
	assert.Equal(t, -6.1751, user.Location.Latitude)
	assert.Equal(t, 106.8650, user.Location.Longitude)
	assert.Equal(t, 2.5, user.Distance)
}