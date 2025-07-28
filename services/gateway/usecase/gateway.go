package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/gateway"
	"github.com/piresc/nebengjek/services/users"
)

// GatewayUseCase implements the gateway use case following clean architecture
type GatewayUseCase struct {
	userUC    users.UserUC
	gatewayGW gateway.GatewayGW
}

// NewGatewayUC creates a new gateway use case with proper dependency injection
func NewGatewayUC(userUC users.UserUC, gatewayGW gateway.GatewayGW) gateway.GatewayUC {
	return &GatewayUseCase{
		userUC:    userUC,
		gatewayGW: gatewayGW,
	}
}

// UpdateBeaconStatus updates beacon status through users service
func (uc *GatewayUseCase) UpdateBeaconStatus(ctx context.Context, req *models.BeaconRequest) error {
	// Call Users Service which will handle business logic and publish to NATS
	resp, err := uc.gatewayGW.CallUsersService(
		ctx,
		"POST",
		"/beacon/update",
		req,
		nil,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to update beacon status: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("users service returned error: %s", string(resp.Body))
	}

	return nil
}

// UpdateFinderStatus updates finder status through users service
func (uc *GatewayUseCase) UpdateFinderStatus(ctx context.Context, req *models.FinderRequest) error {
	// Call Users Service which will handle business logic and publish to NATS
	resp, err := uc.gatewayGW.CallUsersService(
		ctx,
		"POST",
		"/finder/update",
		req,
		nil,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to update finder status: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("users service returned error: %s", string(resp.Body))
	}

	return nil
}

// ConfirmMatch confirms a match through match service
func (uc *GatewayUseCase) ConfirmMatch(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
	// Call Match Service instead of Users Service
	resp, err := uc.gatewayGW.CallMatchService(
		ctx,
		"POST",
		"/matches/confirm",
		req,
		map[string]string{"X-User-ID": req.UserID},
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to confirm match: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("match service returned error: %s", string(resp.Body))
	}

	// Parse response
	var result models.MatchProposal
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse match service response: %w", err)
	}

	return &result, nil
}

// UpdateUserLocation updates user location through location service
func (uc *GatewayUseCase) UpdateUserLocation(ctx context.Context, req *models.LocationUpdate) error {
	// Call Location Service instead of Users Service
	resp, err := uc.gatewayGW.CallLocationService(
		ctx,
		"POST",
		"/locations/update",
		req,
		nil,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to update location: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("location service returned error: %s", string(resp.Body))
	}

	return nil
}

// RideStart starts a ride through rides service
func (uc *GatewayUseCase) RideStart(ctx context.Context, req *models.RideStartRequest) (*models.Ride, error) {
	// Call Rides Service instead of Users Service
	resp, err := uc.gatewayGW.CallRidesService(
		ctx,
		"POST",
		"/rides/start",
		req,
		nil,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to start ride: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rides service returned error: %s", string(resp.Body))
	}

	// Parse response
	var result models.Ride
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rides service response: %w", err)
	}

	return &result, nil
}

// RideArrived handles ride arrival through rides service
func (uc *GatewayUseCase) RideArrived(ctx context.Context, req *models.RideArrivalReq) (*models.PaymentRequest, error) {
	// Call Rides Service instead of Users Service
	resp, err := uc.gatewayGW.CallRidesService(
		ctx,
		"POST",
		"/rides/arrived",
		req,
		nil,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to process ride arrival: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rides service returned error: %s", string(resp.Body))
	}

	// Parse response
	var result models.PaymentRequest
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rides service response: %w", err)
	}

	return &result, nil
}

// ProcessPayment processes payment through rides service
func (uc *GatewayUseCase) ProcessPayment(ctx context.Context, req *models.PaymentProccessRequest) (*models.Payment, error) {
	// Call Rides Service instead of Users Service
	resp, err := uc.gatewayGW.CallRidesService(
		ctx,
		"POST",
		"/rides/payment",
		req,
		nil,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rides service returned error: %s", string(resp.Body))
	}

	// Parse response
	var result models.Payment
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rides service response: %w", err)
	}

	return &result, nil
}

// Proxy operations using Gateway interface
func (uc *GatewayUseCase) ProxyToUsersService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return uc.gatewayGW.CallUsersService(ctx, method, path, body, headers, queryParams)
}

func (uc *GatewayUseCase) ProxyToMatchService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return uc.gatewayGW.CallMatchService(ctx, method, path, body, headers, queryParams)
}

func (uc *GatewayUseCase) ProxyToRidesService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return uc.gatewayGW.CallRidesService(ctx, method, path, body, headers, queryParams)
}

func (uc *GatewayUseCase) ProxyToLocationService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return uc.gatewayGW.CallLocationService(ctx, method, path, body, headers, queryParams)
}
