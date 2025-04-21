package user

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserRepo defines the user repository interface
type UserRepo interface {
	// User management
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByMSISDN(ctx context.Context, msisdn string) (*models.User, error)
	UpdateToDriver(ctx context.Context, user *models.User) error
	// OTP management
	CreateOTP(ctx context.Context, otp *models.OTP) error
	GetOTP(ctx context.Context, msisdn, code string) (*models.OTP, error)
	MarkOTPVerified(ctx context.Context, msisdn string, code string) error
}
