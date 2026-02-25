package users

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/models/user"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/users UserRepo

// UserRepo defines the user repository interface for database operations
type UserRepo interface {
	// User management - PostgreSQL operations
	CreateUser(ctx context.Context, user *user.User) error
	GetUserByID(ctx context.Context, id string) (*user.User, error)
	GetUserByMSISDN(ctx context.Context, msisdn string) (*user.User, error)
	UpdateToDriver(ctx context.Context, user *user.User) error

	// OTP management
	CreateOTP(ctx context.Context, otp *core.OTP) error
	GetOTP(ctx context.Context, msisdn, code string) (*core.OTP, error)
	MarkOTPVerified(ctx context.Context, msisdn string, code string) error

	// Cache operations - Redis operations
	SetEventCacheWithTTL(ctx context.Context, eventType, userID string, req interface{}, ttl time.Duration) (bool, error)
}
