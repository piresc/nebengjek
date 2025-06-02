package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/rides"
)

// RidesHandler handles HTTP requests for ride operations
type RidesHandler struct {
	rideUC rides.RideUC
}

// NewRidesHandler creates a new ride HTTP handler
func NewRidesHandler(rideUC rides.RideUC) *RidesHandler {
	return &RidesHandler{
		rideUC: rideUC,
	}
}

// StartTrip handles the start trip request for a ride
func (h *RidesHandler) StartRide(c echo.Context) error {
	rideID := c.Param("rideID")

	if rideID == "" {
		return utils.BadRequestResponse(c, "Ride ID is required")
	}

	var req models.RideStartRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request body: "+err.Error())
	}

	// Ensure the request contains the ride ID
	req.RideID = rideID

	// Validate request
	if req.DriverLocation == nil || req.PassengerLocation == nil {
		return utils.BadRequestResponse(c, "Driver and passenger locations are required")
	}

	// Call the use case
	resp, err := h.rideUC.StartRide(c.Request().Context(), req)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to start trip: "+err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Trip started successfully", resp)
}
