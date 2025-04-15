package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/user"
)

// BeaconHandler handles beacon-related HTTP requests
type BeaconHandler struct {
	userUC user.UserUC
}

// NewBeaconHandler creates a new beacon handler
func NewBeaconHandler(userUC user.UserUC) *BeaconHandler {
	return &BeaconHandler{
		userUC: userUC,
	}
}

// UpdateBeacon handles beacon status update requests
func (h *BeaconHandler) UpdateBeacon(c echo.Context) error {
	var request models.BeaconRequest
	if err := c.Bind(&request); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Update beacon status
	if err := h.userUC.UpdateBeaconStatus(c.Request().Context(), &request); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Beacon status updated successfully", nil)
}

// GetBeaconStatus handles beacon status retrieval requests
func (h *BeaconHandler) GetBeaconStatus(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return utils.BadRequestResponse(c, "User ID is required")
	}

	// Get user to check beacon status
	user, err := h.userUC.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Beacon status retrieved", map[string]interface{}{
		"is_active": user.IsActive,
		"role":      user.Role,
	})
}
