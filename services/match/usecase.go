package match

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchUC defines the interface for match business logic
type MatchUC interface {
	HandleBeaconEvent(event models.BeaconEvent) error
}
