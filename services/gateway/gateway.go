package gateway

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/models/ride"
	"github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/gateway GatewayUC
//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/gateway GatewayGW

// GatewayUC defines the gateway use case interface
type GatewayUC interface {
	// WebSocket event handlers
	UpdateBeaconStatus(ctx context.Context, req *core.BeaconRequest) error
	UpdateFinderStatus(ctx context.Context, req *core.FinderRequest) error
	ConfirmMatch(ctx context.Context, req *match.MatchConfirmRequest) (*match.MatchProposal, error)
	UpdateUserLocation(ctx context.Context, req *location.LocationUpdate) error
	RideStart(ctx context.Context, req *ride.RideStartRequest) (*ride.Ride, error)
	RideArrived(ctx context.Context, req *ride.RideArrivalReq) (*ride.PaymentRequest, error)
	ProcessPayment(ctx context.Context, req *ride.PaymentProccessRequest) (*ride.Payment, error)

	// Proxy operations
	ProxyToUsersService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
	ProxyToMatchService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
	ProxyToRidesService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
	ProxyToLocationService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
}

// GatewayGW defines the gateway interface for external service communication
type GatewayGW interface {
	// HTTP Gateway operations (Microservice communication)
	CallUsersService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
	CallMatchService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
	CallRidesService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
	CallLocationService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*core.ProxyResponse, error)
}
