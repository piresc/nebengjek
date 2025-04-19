package location

import "github.com/piresc/nebengjek/internal/pkg/models"

// LocationUseCase defines the interface for location business logic
type LocationUC interface {
	StoreLocation(location models.LocationUpdate) error
}
