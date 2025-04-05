package user

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserUC defines the interface for user business logic operations
type UserUC interface {
	// User operations
	RegisterUser(ctx context.Context, user *models.User) error
	AuthenticateUser(ctx context.Context, email, password string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, user *models.User) error
	DeactivateUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error)

	// Driver-specific operations
	RegisterDriver(ctx context.Context, user *models.User) error
	UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error
	UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error
	GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error)
	VerifyDriver(ctx context.Context, driverID string) error
}
