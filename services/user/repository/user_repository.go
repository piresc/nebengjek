package repository

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserRepository defines the interface for user data access operations
type UserRepository interface {
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error)

	// Driver-specific operations
	UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error
	UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error
	GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error)
	VerifyDriver(ctx context.Context, driverID string) error
}
