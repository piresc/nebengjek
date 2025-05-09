package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserUsecase represents the user usecase interface
type UserUC interface {
	RegisterUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)

	// handle OTP
	GenerateOTP(ctx context.Context, msisdn string) error
	VerifyOTP(ctx context.Context, msisdn, otp string) (*models.AuthResponse, error)

	// register driver
	RegisterDriver(ctx context.Context, user *models.User) error

	// handle match
	UpdateBeaconStatus(ctx context.Context, beaconReq *models.BeaconRequest) error
	ConfirmMatch(ctx context.Context, mp *models.MatchProposal, userID string) error

	// handle location
	UpdateUserLocation(ctx context.Context, location *models.LocationUpdate) error

	RideArrived(ctx context.Context, event *models.RideCompleteEvent) error
}
