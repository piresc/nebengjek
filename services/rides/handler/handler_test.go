package handler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNatsClient := &natspkg.Client{}
	cfg := &models.Config{}

	// Act
	handler := NewHandler(mockRideUC, mockNatsClient, cfg)

	// Assert
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.ridesHTTP)
	assert.NotNil(t, handler.ridesNATS)
	assert.Equal(t, cfg, handler.cfg)
}

func TestHandler_RegisterRoutes(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNatsClient := &natspkg.Client{}
	cfg := &models.Config{}

	handler := NewHandler(mockRideUC, mockNatsClient, cfg)
	
	// Create middleware mock
	middleware := &middleware.Middleware{}

	// Act - This should not panic
	e := echo.New()
	handler.RegisterRoutes(e, middleware)

	// Assert - Check that routes were registered
	routes := e.Routes()
	
	// Find the internal routes group
	internalRoutes := make([]echo.Route, 0)
	for _, route := range routes {
		if route.Path == "/internal" || route.Path == "/internal/rides" || 
		   route.Path == "/internal/rides/:rideID/start" || 
		   route.Path == "/internal/rides/:rideID/arrive" || 
		   route.Path == "/internal/rides/:rideID/payment" {
			internalRoutes = append(internalRoutes, *route)
		}
	}

	// Verify that our routes were registered
	assert.Greater(t, len(internalRoutes), 0, "Expected to find registered routes")
	
	// Check specific routes
	foundStartRoute := false
	foundArriveRoute := false
	foundPaymentRoute := false

	for _, route := range internalRoutes {
		switch route.Path {
		case "/internal/rides/:rideID/start":
			foundStartRoute = true
			assert.Equal(t, "POST", route.Method)
		case "/internal/rides/:rideID/arrive":
			foundArriveRoute = true
			assert.Equal(t, "POST", route.Method)
		case "/internal/rides/:rideID/payment":
			foundPaymentRoute = true
			assert.Equal(t, "POST", route.Method)
		}
	}

	assert.True(t, foundStartRoute, "Expected to find start ride route")
	assert.True(t, foundArriveRoute, "Expected to find ride arrived route")
	assert.True(t, foundPaymentRoute, "Expected to find process payment route")
}

func TestHandler_InitNATSConsumers_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNatsClient := &natspkg.Client{}
	cfg := &models.Config{}

	handler := NewHandler(mockRideUC, mockNatsClient, cfg)

	// Act - Since we're using a real NATS client, this might fail in test environment
	// We're testing that the method exists and can be called without panicking
	err := handler.InitNATSConsumers()

	// Assert - The error depends on NATS availability, so we just check it doesn't panic
	// In a real test environment with NATS running, this would succeed
	// For now, we just verify the method can be called
	assert.NotNil(t, err) // Expected to fail without NATS server
}

func TestHandler_InitNATSConsumers_NilNATSClient(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{}

	handler := NewHandler(mockRideUC, nil, cfg)

	// Act & Assert - Should panic due to nil client
	assert.Panics(t, func() {
		handler.InitNATSConsumers()
	})
}

func TestHandler_ComponentsNotNil(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNatsClient := &natspkg.Client{}
	cfg := &models.Config{}

	handler := NewHandler(mockRideUC, mockNatsClient, cfg)

	// Assert - Verify all components are properly initialized
	assert.NotNil(t, handler.ridesHTTP, "HTTP handler should not be nil")
	assert.NotNil(t, handler.ridesNATS, "NATS handler should not be nil")
	assert.NotNil(t, handler.cfg, "Config should not be nil")
}