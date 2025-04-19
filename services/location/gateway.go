package location

import (
	"context"
)

// LocationGW defines the interface for location gateway operations
type LocationGW interface {
	// PublishLocationAggregate publishes a location aggregate event to NATS
	PublishLocationAggregate(ctx context.Context, aggregate interface{}) error
}
