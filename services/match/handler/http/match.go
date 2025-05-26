package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/match"
)

// MatchHandler handles HTTP requests for match operations
type MatchHandler struct {
	matchUC match.MatchUC
}

// NewMatchHandler creates a new match HTTP handler
func NewMatchHandler(matchUC match.MatchUC) *MatchHandler {
	return &MatchHandler{
		matchUC: matchUC,
	}
}

// RegisterRoutes registers the match handler routes
func (h *MatchHandler) RegisterRoutes(e *echo.Echo) {
	matchGroup := e.Group("/matches")
	matchGroup.POST("/:matchID/confirm", h.ConfirmMatch)
}

// MatchConfirmRequest is the request structure for match confirmation
type MatchConfirmRequest struct {
	UserID string             `json:"userId"`
	Status models.MatchStatus `json:"status"`
}

// MatchConfirmResponse is the response structure for match confirmation
type MatchConfirmResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	MatchID string               `json:"matchId,omitempty"`
	Match   models.MatchProposal `json:"match,omitempty"`
}

// ConfirmMatch handles the confirmation of a match by a user
func (h *MatchHandler) ConfirmMatch(c echo.Context) error {
	matchID := c.Param("matchID")

	if matchID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Match ID is required",
		})
	}

	var req MatchConfirmRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	// Validate request
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "User ID is required",
		})
	}

	if req.Status != models.MatchStatusAccepted && req.Status != models.MatchStatusRejected {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Status must be either ACCEPTED or REJECTED",
		})
	}

	// Create a match proposal with user confirmation
	// We'll use the matchID to look up the actual match details
	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    req.UserID, // The service will use this to determine which user is confirming
		MatchStatus: req.Status,
	}

	// Call the use case to confirm the match
	result, err := h.matchUC.ConfirmMatchStatus(matchID, matchProposal)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to confirm match: " + err.Error(),
		})
	}

	// Return success response with match details
	return c.JSON(http.StatusOK, MatchConfirmResponse{
		Success: true,
		Message: "Match confirmation processed successfully",
		MatchID: matchID,
		Match:   result,
	})
}
