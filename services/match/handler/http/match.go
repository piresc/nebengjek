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

// MatchConfirmResponse is the response structure for match confirmation
type MatchConfirmResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	MatchID string               `json:"match_id,omitempty"` // Changed to match snake_case convention
	Match   models.MatchProposal `json:"match,omitempty"`
}

// ConfirmMatch handles the confirmation of a match by a user
func (h *MatchHandler) ConfirmMatch(c echo.Context) error {
	matchID := c.Param("matchID")

	if matchID == "" {
		return BadRequestResponse(c, "Match ID is required")
	}

	var req models.MatchConfirmRequest
	if err := c.Bind(&req); err != nil {
		return BadRequestResponse(c, "Invalid request body: "+err.Error())
	}
	// Ensure the request contains the match ID
	req.ID = matchID

	// Validate request
	if req.UserID == "" {
		return BadRequestResponse(c, "User ID is required")
	}

	if req.Status != string(models.MatchStatusAccepted) && req.Status != string(models.MatchStatusRejected) {
		return BadRequestResponse(c, "Status must be either ACCEPTED or REJECTED")
	}

	// Call the use case directly - let the usecase handle all validation logic
	result, err := h.matchUC.ConfirmMatchStatus(&req)
	if err != nil {
		return ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to confirm match: "+err.Error())
	}

	// Return success response with match details
	responseData := MatchConfirmResponse{
		Success: true,
		Message: "Match confirmation processed successfully",
		MatchID: matchID,
		Match:   result,
	}

	return SuccessResponseWithData(c, http.StatusOK, "Match confirmation processed successfully", responseData)
}
