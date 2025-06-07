package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
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
	// Get transaction from Echo context using centralized package
	txn := nrpkg.FromEchoContext(c)
	nrpkg.SetTransactionName(txn, "Match.ConfirmMatch")

	matchID := c.Param("matchID")
	if matchID == "" {
		return utils.BadRequestResponse(c, "Match ID is required")
	}

	var req models.MatchConfirmRequest
	if err := c.Bind(&req); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.BadRequestResponse(c, "Invalid request body: "+err.Error())
	}

	req.ID = matchID

	if req.UserID == "" {
		return utils.BadRequestResponse(c, "User ID is required")
	}

	if req.Status != string(models.MatchStatusAccepted) && req.Status != string(models.MatchStatusRejected) {
		return utils.BadRequestResponse(c, "Status must be either ACCEPTED or REJECTED")
	}

	// Add transaction attributes for better tracing
	nrpkg.AddTransactionAttribute(txn, "endpoint", "confirm_match")
	nrpkg.AddTransactionAttribute(txn, "match.id", matchID)
	nrpkg.AddTransactionAttribute(txn, "user.id", req.UserID)
	nrpkg.AddTransactionAttribute(txn, "match.status", req.Status)

	result, err := h.matchUC.ConfirmMatchStatus(c.Request().Context(), &req)
	if err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to confirm match: "+err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Match confirmation processed successfully", result)
}
