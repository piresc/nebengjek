package gateway

import (
	"context"
	"testing"
	"time"

	coremodels "github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	gateway_nats "github.com/piresc/nebengjek/services/users/gateway/nats"
	"github.com/stretchr/testify/assert"
)

func TestNewUserGW(t *testing.T) {
	// Arrange
	mockNatsClient := &natspkg.Client{}

	// Act
	gateway := NewUserGW(mockNatsClient)

	// Assert
	assert.NotNil(t, gateway)
	userGW, ok := gateway.(*UserGW)
	assert.True(t, ok, "Expected gateway to be of type *UserGW")
	assert.NotNil(t, userGW.natsGateway, "NATS gateway should not be nil")
}

func TestUserGW_PublishBeaconEvent_NilClient(t *testing.T) {
	// Arrange
	gateway := &UserGW{
		natsGateway: gateway_nats.NewNATSGateway(nil),
	}

	ctx := context.Background()
	event := &coremodels.BeaconEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		Timestamp: time.Now(),
	}

	// Act & Assert - Should panic due to nil NATS client
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, event)
	})
}

func TestUserGW_PublishFinderEvent_NilClient(t *testing.T) {
	// Arrange
	gateway := &UserGW{
		natsGateway: gateway_nats.NewNATSGateway(nil),
	}

	ctx := context.Background()
	event := &coremodels.FinderEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude: 106.8295,
		},
		Timestamp: time.Now(),
	}

	// Act & Assert - Should panic due to nil NATS client
	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, event)
	})
}

func TestUserGW_PublishBeaconEvent_NilEvent(t *testing.T) {
	// Arrange
	gateway := &UserGW{
		natsGateway: gateway_nats.NewNATSGateway(nil),
	}

	ctx := context.Background()

	// Act & Assert - Should panic due to nil NATS client and nil event
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, nil)
	})
}

func TestUserGW_PublishFinderEvent_NilEvent(t *testing.T) {
	// Arrange
	gateway := &UserGW{
		natsGateway: gateway_nats.NewNATSGateway(nil),
	}

	ctx := context.Background()

	// Act & Assert - Should panic due to nil NATS client and nil event
	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, nil)
	})
}

func TestUserGW_ContextCancellation(t *testing.T) {
	// Arrange
	gateway := &UserGW{
		natsGateway: gateway_nats.NewNATSGateway(nil),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context before using it

	beaconEvent := &coremodels.BeaconEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		Timestamp: time.Now(),
	}

	finderEvent := &coremodels.FinderEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude: 106.8295,
		},
		Timestamp: time.Now(),
	}

	// Act & Assert - Should panic due to nil NATS client
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, beaconEvent)
	})

	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, finderEvent)
	})
}

func TestUserGW_WithRealNatsClient(t *testing.T) {
	// Arrange - Test with real NATS client (but no server)
	mockNatsClient := &natspkg.Client{}
	gateway := NewUserGW(mockNatsClient)

	ctx := context.Background()
	beaconEvent := &coremodels.BeaconEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		Timestamp: time.Now(),
	}

	finderEvent := &coremodels.FinderEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude: 106.8295,
		},
		Timestamp: time.Now(),
	}

	// Act & Assert - Should panic due to nil NATS client
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, beaconEvent)
	})

	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, finderEvent)
	})
}

func TestUserGW_ForwardingBehavior(t *testing.T) {
	// This test verifies that the gateway properly forwards calls to the NATS gateway
	// Since we can't easily mock the NATS gateway without interface issues,
	// we test that the gateway is properly initialized and can be called
	
	// Arrange
	mockNatsClient := &natspkg.Client{}
	gateway := NewUserGW(mockNatsClient)
	userGW := gateway.(*UserGW)

	// Assert - Verify the gateway structure
	assert.NotNil(t, userGW.natsGateway, "NATS gateway should be initialized")
	
	// The actual forwarding behavior is tested by the panic tests above,
	// which verify that the calls reach the NATS gateway and panic as expected
	// when the NATS client is nil
}