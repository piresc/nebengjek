package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/users UserUC

// UserUsecase represents the user usecase interface
type UserUC interface {
	// User operations
	RegisterUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)

	// handle OTP
	GenerateOTP(ctx context.Context, msisdn string) error
	VerifyOTP(ctx context.Context, msisdn, otp string) (*models.AuthResponse, error)

	// register driver
	RegisterDriver(ctx context.Context, user *models.User) error

	UpdateBeaconStatus(ctx context.Context, beaconReq *models.BeaconRequest) error
	UpdateFinderStatus(ctx context.Context, finderReq *models.FinderRequest) error
}
