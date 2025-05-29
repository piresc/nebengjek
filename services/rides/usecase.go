package rides

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideUC defines the interface for ride business logic
// go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/rides RideUC
type RideUC interface {
	CreateRide(mp models.MatchConfirmRequest) error
	ProcessBillingUpdate(rideID string, entry *models.BillingLedger) error
	CompleteRide(rideID string, adjustmentFactor float64) (*models.Payment, error)
}
