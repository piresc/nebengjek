package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/utils"
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

// ConfirmMatch handles the confirmation of a match by a user
func (h *MatchHandler) ConfirmMatch(c echo.Context) error {
	matchID := c.Param("matchID")
	if matchID == "" {
		return utils.BadRequestResponse(c, "Match ID is required")
	}

	var req matchmodels.MatchConfirmRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request body: "+err.Error())
	}

	req.ID = matchID

	if req.UserID == "" {
		return utils.BadRequestResponse(c, "User ID is required")
	}

	if req.Status != string(matchmodels.MatchStatusAccepted) && req.Status != string(matchmodels.MatchStatusRejected) {
		return utils.BadRequestResponse(c, "Status must be either ACCEPTED or REJECTED")
	}

	result, err := h.matchUC.ConfirmMatchStatus(c.Request().Context(), &req)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to confirm match: "+err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Match confirmation processed successfully", result)
}
