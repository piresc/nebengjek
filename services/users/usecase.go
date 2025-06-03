package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/users UserUC

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
	UpdateFinderStatus(ctx context.Context, finderReq *models.FinderRequest) error

	// handle match confirmation
	ConfirmMatch(ctx context.Context, mp *models.MatchConfirmRequest) (*models.MatchProposal, error)

	// handle location
	UpdateUserLocation(ctx context.Context, location *models.LocationUpdate) error

	// handle ride events
	RideStart(ctx context.Context, event *models.RideStartRequest) (*models.Ride, error)
	RideArrived(ctx context.Context, req *models.RideArrivalReq) (*models.PaymentRequest, error)
	ProcessPayment(ctx context.Context, paymentReq *models.PaymentProccessRequest) (*models.Payment, error)
}
