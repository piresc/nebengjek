package location

import (
	"context"

		"github.com/piresc/nebengjek/internal/pkg/models/location"
)

// LocationGW defines the interface for location gateway operations
type LocationGW interface {
	// PublishLocationAggregate publishes a location aggregate event to NATS
	PublishLocationAggregate(ctx context.Context, aggregate location.LocationAggregate) error
}
