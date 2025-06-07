package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location"
	httpHandler "github.com/piresc/nebengjek/services/location/handler/http"
)

// HTTPHandler combines all handlers for the location service
type HTTPHandler struct {
	locationHTTP *httpHandler.LocationHandler
	locationNATS *LocationHandler // This is the existing NATS handler in the same package
	cfg          *models.Config
}

// NewHTTPHandler creates a new combined handler
func NewHTTPHandler(
	locationUC location.LocationUC,
	natsClient *natspkg.Client,
	cfg *models.Config,
	nrApp *newrelic.Application,
) *HTTPHandler {
	return &HTTPHandler{
		locationHTTP: httpHandler.NewLocationHandler(locationUC),
		locationNATS: NewLocationHandler(locationUC, natsClient, nrApp), // Use existing NATS handler
		cfg:          cfg,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *HTTPHandler) RegisterRoutes(e *echo.Echo, unifiedMiddleware *middleware.UnifiedMiddleware) {
	// Internal routes for service-to-service communication (API key required)
	internal := e.Group("/internal", unifiedMiddleware.APIKeyHandler("match-service"))

	// Driver routes
	internal.POST("/drivers/:id/available", h.locationHTTP.AddAvailableDriver)
	internal.DELETE("/drivers/:id/available", h.locationHTTP.RemoveAvailableDriver)
	internal.GET("/drivers/:id/location", h.locationHTTP.GetDriverLocation)
	internal.GET("/drivers/nearby", h.locationHTTP.FindNearbyDrivers)

	// Passenger routes
	internal.POST("/passengers/:id/available", h.locationHTTP.AddAvailablePassenger)
	internal.DELETE("/passengers/:id/available", h.locationHTTP.RemoveAvailablePassenger)
	internal.GET("/passengers/:id/location", h.locationHTTP.GetPassengerLocation)
}

// InitNATSConsumers initializes all NATS consumers
func (h *HTTPHandler) InitNATSConsumers() error {
	return h.locationNATS.InitNATSConsumers()
}
