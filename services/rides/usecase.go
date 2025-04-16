package rides

import "github.com/piresc/nebengjek/internal/pkg/models"

// RideUC defines the interface for ride use cases
type RideUC interface {
	// CreateRide creates a new ride from a confirmed match
	CreateRide(ride models.MatchProposal) error
}
