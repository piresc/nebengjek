package rides

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideUC defines the interface for ride business logic
type RideUC interface {
	CreateRide(mp models.MatchProposal) error
	ProcessBillingUpdate(rideID string, entry *models.BillingLedger) error
}
