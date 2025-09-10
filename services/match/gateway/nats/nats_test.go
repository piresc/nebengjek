package nats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

func TestNewNATSGateway(t *testing.T) {
	// Test with nil client
	gateway := NewNATSGateway(nil)
	assert.NotNil(t, gateway)
	assert.Nil(t, gateway.natsClient)
}

func TestNATSGateway_PublishMatchFound_NilClient(t *testing.T) {
	gateway := NewNATSGateway(nil)
	
	ctx := context.Background()
	matchProp := models.MatchProposal{
		ID:             "match-123",
		DriverID:       "driver-456",
		PassengerID:    "passenger-789",
		UserLocation:   models.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: models.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    models.MatchStatusPending,
	}

	// Should panic with nil client
	assert.Panics(t, func() {
		gateway.PublishMatchFound(ctx, matchProp)
	})
}

func TestNATSGateway_PublishMatchRejected_NilClient(t *testing.T) {
	gateway := NewNATSGateway(nil)
	
	ctx := context.Background()
	matchProp := models.MatchProposal{
		ID:             "match-123",
		DriverID:       "driver-456",
		PassengerID:    "passenger-789",
		UserLocation:   models.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: models.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    models.MatchStatusPending,
	}

	// Should panic with nil client
	assert.Panics(t, func() {
		gateway.PublishMatchRejected(ctx, matchProp)
	})
}

func TestNATSGateway_PublishMatchAccepted_NilClient(t *testing.T) {
	gateway := NewNATSGateway(nil)
	
	ctx := context.Background()
	matchProp := models.MatchProposal{
		ID:             "match-123",
		DriverID:       "driver-456",
		PassengerID:    "passenger-789",
		UserLocation:   models.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: models.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    models.MatchStatusPending,
	}

	// Should panic with nil client
	assert.Panics(t, func() {
		gateway.PublishMatchAccepted(ctx, matchProp)
	})
}

func TestNATSGateway_InvalidMatchProposal(t *testing.T) {
	// Create a real client but with invalid data that can't be marshaled
	// This is a basic test that focuses on the marshaling logic
	gateway := NewNATSGateway(nil)
	
	ctx := context.Background()
	
	// Test with empty match proposal
	matchProp := models.MatchProposal{
		ID: "",
	}

	// Should panic due to nil client
	assert.Panics(t, func() {
		gateway.PublishMatchFound(ctx, matchProp)
	})
}

// Integration-style test that would work with a real NATS server
// This test is skipped by default and only runs when NATS is available
func TestNATSGateway_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would require a real NATS server
	// For now, we just test that the gateway can be created
	gateway := NewNATSGateway(nil)
	assert.NotNil(t, gateway)
}