package rides

import "github.com/piresc/nebengjek/internal/pkg/models"

// RideRepo defines the interface for ride data access operations
type RideRepo interface {
	CreateRide(ride *models.Ride) (*models.Ride, error)
}
