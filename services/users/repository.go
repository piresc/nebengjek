package users

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/users UserRepo

// UserRepo defines the user repository interface for database operations
type UserRepo interface {
	// User management - PostgreSQL operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByMSISDN(ctx context.Context, msisdn string) (*models.User, error)
	UpdateToDriver(ctx context.Context, user *models.User) error

	// OTP management
	CreateOTP(ctx context.Context, otp *models.OTP) error
	GetOTP(ctx context.Context, msisdn, code string) (*models.OTP, error)
	MarkOTPVerified(ctx context.Context, msisdn string, code string) error

	// Cache operations - Redis operations
	SetEventCacheWithTTL(ctx context.Context, eventType, userID string, req interface{}, ttl time.Duration) (bool, error)
}
