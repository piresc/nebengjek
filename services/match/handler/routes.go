package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
	httpHandler "github.com/piresc/nebengjek/services/match/handler/http"
	natsHandler "github.com/piresc/nebengjek/services/match/handler/nats"
)

// Handler combines all handlers for the match service
type Handler struct {
	matchHTTP *httpHandler.MatchHandler
	matchNATS *natsHandler.MatchHandler
}

// NewHandler creates a new combined handler
func NewHandler(
	matchUC match.MatchUC,
	natsClient *natspkg.Client,
	nrApp *newrelic.Application,
) *Handler {
	return &Handler{
		matchHTTP: httpHandler.NewMatchHandler(matchUC),
		matchNATS: natsHandler.NewMatchHandler(matchUC, natsClient, nrApp),
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(e *echo.Echo, apiKeyMiddleware *middleware.APIKeyMiddleware) {
	// Internal routes for service-to-service communication (API key required)
	internal := e.Group("/internal", apiKeyMiddleware.ValidateAPIKey("match-service"))

	// Internal match endpoints
	internalMatchGroup := internal.Group("/matches")
	internalMatchGroup.POST("/:matchID/confirm", h.matchHTTP.ConfirmMatch)
}

// InitNATSConsumers initializes all NATS consumers
func (h *Handler) InitNATSConsumers() error {
	return h.matchNATS.InitNATSConsumers()
}
