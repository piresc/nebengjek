package user

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserUsecase represents the user usecase interface
type UserUC interface {
	RegisterUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, user *models.User) error
	DeactivateUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error)
	GenerateOTP(ctx context.Context, msisdn string) error
	VerifyOTP(ctx context.Context, msisdn, otp string) (*models.AuthResponse, error)
	RegisterDriver(ctx context.Context, user *models.User) error
	UpdateBeaconStatus(ctx context.Context, beaconReq *models.BeaconRequest) error
}
