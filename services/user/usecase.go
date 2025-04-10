package user

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserUC defines the user use case interface
type UserUC interface {
	// User management
	RegisterUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, user *models.User) error
	DeactivateUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error)

	// Authentication with OTP
	GenerateOTP(ctx context.Context, msisdn string) error
	VerifyOTP(ctx context.Context, msisdn, otp string) (*models.AuthResponse, error)

	// Driver management
	RegisterDriver(ctx context.Context, user *models.User) error
}
