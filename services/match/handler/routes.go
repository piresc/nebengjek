package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/services/match"
)

// MatchHandler handles HTTP requests for the match service
type MatchHandler struct {
	matchUC match.MatchUseCase
}

// NewMatchHandler creates a new match handler
func NewMatchHandler(matchUC match.MatchUseCase) *MatchHandler {
	return &MatchHandler{
		matchUC: matchUC,
	}
}

// RegisterRoutes registers the match service routes
func (h *MatchHandler) RegisterRoutes(e *echo.Echo) {
	// Match endpoints
	e.POST("/matches", h.CreateMatchRequest)
	e.GET("/matches/driver/:id", h.GetDriverMatches)
	e.GET("/matches/passenger/:id", h.GetPassengerMatches)
	e.PUT("/matches/:id/accept", h.AcceptMatch)
	e.PUT("/matches/:id/reject", h.RejectMatch)
	e.PUT("/matches/:id/cancel", h.CancelMatch)
}
