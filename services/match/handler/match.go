package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// CreateMatchRequest handles match request creation
func (h *MatchHandler) CreateMatchRequest(c echo.Context) error {
	// Parse request body
	var request struct {
		PassengerID     string          `json:"passenger_id"`
		PickupLocation  models.Location `json:"pickup_location"`
		DropoffLocation models.Location `json:"dropoff_location"`
	}

	if err := c.Bind(&request); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Invalid request format")
	}

	// Validate request
	if request.PassengerID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Passenger ID is required")
	}

	// Create trip object
	trip := &models.Trip{
		PassengerID:     request.PassengerID,
		PickupLocation:  request.PickupLocation,
		DropoffLocation: request.DropoffLocation,
		Status:          models.TripStatusRequested,
	}

	// Create match request
	if err := h.matchUC.CreateMatchRequest(c.Request().Context(), trip); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to create match request")
	}

	return utils.SuccessResponse(c, http.StatusCreated, "Match request created successfully", map[string]string{
		"trip_id": trip.ID,
	})
}

// GetDriverMatches retrieves pending matches for a driver
func (h *MatchHandler) GetDriverMatches(c echo.Context) error {
	driverID := c.Param("id")
	if driverID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Driver ID is required")
	}

	// Get pending matches for the driver
	matches, err := h.matchUC.GetPendingMatchesForDriver(c.Request().Context(), driverID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to get driver matches")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Driver matches retrieved successfully", matches)
}

// GetPassengerMatches retrieves pending matches for a passenger
func (h *MatchHandler) GetPassengerMatches(c echo.Context) error {
	passengerID := c.Param("id")
	if passengerID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Passenger ID is required")
	}

	// Get pending matches for the passenger
	matches, err := h.matchUC.GetPendingMatchesForPassenger(c.Request().Context(), passengerID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to get passenger matches")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Passenger matches retrieved successfully", matches)
}

// AcceptMatch handles match acceptance by a driver
func (h *MatchHandler) AcceptMatch(c echo.Context) error {
	tripID := c.Param("id")
	if tripID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Trip ID is required")
	}

	// Parse request body
	var request struct {
		DriverID string `json:"driver_id"`
	}

	if err := c.Bind(&request); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Invalid request format")
	}

	// Validate request
	if request.DriverID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Driver ID is required")
	}

	// Accept the match
	if err := h.matchUC.AcceptMatch(c.Request().Context(), tripID, request.DriverID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to accept match")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Match accepted successfully", nil)
}

// RejectMatch handles match rejection by a driver
func (h *MatchHandler) RejectMatch(c echo.Context) error {
	tripID := c.Param("id")
	if tripID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Trip ID is required")
	}

	// Parse request body
	var request struct {
		DriverID string `json:"driver_id"`
	}

	if err := c.Bind(&request); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Invalid request format")
	}

	// Validate request
	if request.DriverID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Driver ID is required")
	}

	// Reject the match
	if err := h.matchUC.RejectMatch(c.Request().Context(), tripID, request.DriverID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to reject match")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Match rejected successfully", nil)
}

// CancelMatch handles match cancellation by a user (passenger or driver)
func (h *MatchHandler) CancelMatch(c echo.Context) error {
	tripID := c.Param("id")
	if tripID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Trip ID is required")
	}

	// Parse request body
	var request struct {
		UserID string `json:"user_id"`
	}

	if err := c.Bind(&request); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "Invalid request format")
	}

	// Validate request
	if request.UserID == "" {
		return utils.ErrorResponseHandler(c, http.StatusBadRequest, "User ID is required")
	}

	// Cancel the match
	if err := h.matchUC.CancelMatch(c.Request().Context(), tripID, request.UserID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to cancel match")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Match cancelled successfully", nil)
}
