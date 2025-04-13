package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// NATSProducer interface for publishing messages to NATS
type NATSProducer interface {
	Publish(topic string, message interface{}) error
}

// ToggleBeacon handles requests to toggle a user's beacon status
func (h *UserHandler) ToggleBeacon(c echo.Context) error {
	// Parse request
	var request models.BeaconRequest
	if err := c.Bind(&request); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Update beacon status and publish event
	if err := h.userUC.UpdateBeaconStatus(c.Request().Context(), &request); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Beacon status updated", nil)
}
