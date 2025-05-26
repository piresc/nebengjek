package handler

import (
	"github.com/labstack/echo/v4"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
	httpHandler "github.com/piresc/nebengjek/services/match/handler/http"
)

// Handler combines all handlers for the match service
type Handler struct {
	matchHTTP *httpHandler.MatchHandler
	matchNATS *MatchHandler
}

// NewHandler creates a new combined handler
func NewHandler(
	matchUC match.MatchUC,
	natsClient *natspkg.Client,
) *Handler {
	return &Handler{
		matchHTTP: httpHandler.NewMatchHandler(matchUC),
		matchNATS: NewMatchHandler(matchUC, natsClient),
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	h.matchHTTP.RegisterRoutes(e)
}

// InitNATSConsumers initializes all NATS consumers
func (h *Handler) InitNATSConsumers() error {
	return h.matchNATS.InitNATSConsumers()
}
