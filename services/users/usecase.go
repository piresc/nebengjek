package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/models/user"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/users UserUC

// UserUsecase represents the user usecase interface
type UserUC interface {
	// User operations
	RegisterUser(ctx context.Context, user *user.User) error
	GetUserByID(ctx context.Context, id string) (*user.User, error)

	// handle OTP
	GenerateOTP(ctx context.Context, msisdn string) error
	VerifyOTP(ctx context.Context, msisdn, otp string) (*core.AuthResponse, error)

	// register driver
	RegisterDriver(ctx context.Context, user *user.User) error

	UpdateBeaconStatus(ctx context.Context, beaconReq *core.BeaconRequest) error
	UpdateFinderStatus(ctx context.Context, finderReq *core.FinderRequest) error
}
