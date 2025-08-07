package gateway

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/gateway GatewayUC
//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/gateway GatewayGW

// GatewayUC defines the gateway use case interface
type GatewayUC interface {
	// WebSocket event handlers
	UpdateBeaconStatus(ctx context.Context, req *models.BeaconRequest) error
	UpdateFinderStatus(ctx context.Context, req *models.FinderRequest) error
	ConfirmMatch(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchProposal, error)
	UpdateUserLocation(ctx context.Context, req *models.LocationUpdate) error
	RideStart(ctx context.Context, req *models.RideStartRequest) (*models.Ride, error)
	RideArrived(ctx context.Context, req *models.RideArrivalReq) (*models.PaymentRequest, error)
	ProcessPayment(ctx context.Context, req *models.PaymentProccessRequest) (*models.Payment, error)

	// Proxy operations
	ProxyToUsersService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
	ProxyToMatchService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
	ProxyToRidesService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
	ProxyToLocationService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
}

// GatewayGW defines the gateway interface for external service communication
type GatewayGW interface {
	// HTTP Gateway operations (Microservice communication)
	CallUsersService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
	CallMatchService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
	CallRidesService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
	CallLocationService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error)
}
