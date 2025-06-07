package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides"
	httpHandler "github.com/piresc/nebengjek/services/rides/handler/http"
	natsHandler "github.com/piresc/nebengjek/services/rides/handler/nats"
)

// Handler combines all handlers for the rides service
type Handler struct {
	ridesHTTP *httpHandler.RidesHandler
	ridesNATS *natsHandler.RidesHandler
	cfg       *models.Config
}

// NewHandler creates a new combined handler
func NewHandler(
	ridesUC rides.RideUC,
	natsClient *natspkg.Client,
	cfg *models.Config,
	nrApp *newrelic.Application,
) *Handler {
	return &Handler{
		ridesHTTP: httpHandler.NewRidesHandler(ridesUC),
		ridesNATS: natsHandler.NewRidesHandler(ridesUC, natsClient, cfg, nrApp),
		cfg:       cfg,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(e *echo.Echo, Middleware *middleware.Middleware) {
	// Internal routes for service-to-service communication (API key required)
	internal := e.Group("/internal", Middleware.APIKeyHandler("rides-service"))

	// Internal rides endpoints
	internalRidesGroup := internal.Group("/rides")
	internalRidesGroup.POST("/:rideID/start", h.ridesHTTP.StartRide)
	internalRidesGroup.POST("/:rideID/arrive", h.ridesHTTP.RideArrived)
	internalRidesGroup.POST("/:rideID/payment", h.ridesHTTP.ProcessPayment)
}

// InitNATSConsumers initializes all NATS consumers
func (h *Handler) InitNATSConsumers() error {
	return h.ridesNATS.InitNATSConsumers()
}
