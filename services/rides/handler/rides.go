package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// CreateRideRequest handles ride request creation
func (h *RideHandler) CreateRideRequest(c echo.Context) error {
	// Parse request body
	type RideRequestBody struct {
		PassengerID     string          `json:"passenger_id"`
		PickupLocation  models.Location `json:"pickup_location"`
		DropoffLocation models.Location `json:"dropoff_location"`
	}

	var req RideRequestBody
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate request
	if req.PassengerID == "" {
		return utils.BadRequestResponse(c, "Passenger ID is required")
	}

	// Create ride request
	trip, err := h.RideUC.CreateRideRequest(
		c.Request().Context(),
		req.PassengerID,
		&req.PickupLocation,
		&req.DropoffLocation,
	)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusCreated, "Ride request created successfully", trip)
}

// CancelRideRequest handles ride request cancellation
func (h *RideHandler) CancelRideRequest(c echo.Context) error {
	// Get trip ID from path parameter
	tripID := c.Param("id")
	if tripID == "" {
		return utils.BadRequestResponse(c, "Trip ID is required")
	}

	// Get user ID from query parameter
	userID := c.QueryParam("user_id")
	if userID == "" {
		return utils.BadRequestResponse(c, "User ID is required")
	}

	// Cancel ride request
	if err := h.RideUC.CancelRideRequest(c.Request().Context(), tripID, userID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride request cancelled successfully", nil)
}

// AcceptRide handles ride acceptance by a driver
func (h *RideHandler) AcceptRide(c echo.Context) error {
	// Get trip ID from path parameter
	tripID := c.Param("id")
	if tripID == "" {
		return utils.BadRequestResponse(c, "Trip ID is required")
	}

	// Parse request body
	type AcceptRideBody struct {
		DriverID string `json:"driver_id"`
	}

	var req AcceptRideBody
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate request
	if req.DriverID == "" {
		return utils.BadRequestResponse(c, "Driver ID is required")
	}

	// Accept ride
	if err := h.RideUC.AcceptRide(c.Request().Context(), tripID, req.DriverID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride accepted successfully", nil)
}

// RejectRide handles ride rejection by a driver
func (h *RideHandler) RejectRide(c echo.Context) error {
	// Get trip ID from path parameter
	tripID := c.Param("id")
	if tripID == "" {
		return utils.BadRequestResponse(c, "Trip ID is required")
	}

	// Parse request body
	type RejectRideBody struct {
		DriverID string `json:"driver_id"`
	}

	var req RejectRideBody
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate request
	if req.DriverID == "" {
		return utils.BadRequestResponse(c, "Driver ID is required")
	}

	// Reject ride
	if err := h.RideUC.RejectRide(c.Request().Context(), tripID, req.DriverID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride rejected successfully", nil)
}

// StartRide handles ride start by a driver
func (h *RideHandler) StartRide(c echo.Context) error {
	// Get trip ID from path parameter
	tripID := c.Param("id")
	if tripID == "" {
		return utils.BadRequestResponse(c, "Trip ID is required")
	}

	// Parse request body
	type StartRideBody struct {
		DriverID string `json:"driver_id"`
	}

	var req StartRideBody
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate request
	if req.DriverID == "" {
		return utils.BadRequestResponse(c, "Driver ID is required")
	}

	// Start ride
	if err := h.RideUC.StartRide(c.Request().Context(), tripID, req.DriverID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride started successfully", nil)
}

// CompleteRide handles ride completion by a driver
func (h *RideHandler) CompleteRide(c echo.Context) error {
	// Get trip ID from path parameter
	tripID := c.Param("id")
	if tripID == "" {
		return utils.BadRequestResponse(c, "Trip ID is required")
	}

	// Parse request body
	type CompleteRideBody struct {
		DriverID string `json:"driver_id"`
	}

	var req CompleteRideBody
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate request
	if req.DriverID == "" {
		return utils.BadRequestResponse(c, "Driver ID is required")
	}

	// Complete ride
	if err := h.RideUC.CompleteRide(c.Request().Context(), tripID, req.DriverID); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	// Get updated ride with fare
	trip, err := h.RideUC.GetRideStatus(c.Request().Context(), tripID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride completed successfully", trip)
}

// GetRideStatus handles ride status retrieval
func (h *RideHandler) GetRideStatus(c echo.Context) error {
	// Get trip ID from path parameter
	tripID := c.Param("id")
	if tripID == "" {
		return utils.BadRequestResponse(c, "Trip ID is required")
	}

	// Get ride status
	trip, err := h.RideUC.GetRideStatus(c.Request().Context(), tripID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride status retrieved successfully", trip)
}

// GetActiveRideForPassenger handles active ride retrieval for a passenger
func (h *RideHandler) GetActiveRideForPassenger(c echo.Context) error {
	// Get passenger ID from path parameter
	passengerID := c.Param("id")
	if passengerID == "" {
		return utils.BadRequestResponse(c, "Passenger ID is required")
	}

	// Get active ride
	trip, err := h.RideUC.GetActiveRideForPassenger(c.Request().Context(), passengerID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	if trip == nil {
		return utils.SuccessResponse(c, http.StatusOK, "No active ride found", nil)
	}

	return utils.SuccessResponse(c, http.StatusOK, "Active ride retrieved successfully", trip)
}

// GetActiveRideForDriver handles active ride retrieval for a driver
func (h *RideHandler) GetActiveRideForDriver(c echo.Context) error {
	// Get driver ID from path parameter
	driverID := c.Param("id")
	if driverID == "" {
		return utils.BadRequestResponse(c, "Driver ID is required")
	}

	// Get active ride
	trip, err := h.RideUC.GetActiveRideForDriver(c.Request().Context(), driverID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	if trip == nil {
		return utils.SuccessResponse(c, http.StatusOK, "No active ride found", nil)
	}

	return utils.SuccessResponse(c, http.StatusOK, "Active ride retrieved successfully", trip)
}

// GetRideHistory handles ride history retrieval
func (h *RideHandler) GetRideHistory(c echo.Context) error {
	// Get parameters from query string
	userID := c.QueryParam("user_id")
	role := c.QueryParam("role")
	startTimeStr := c.QueryParam("start_time")
	endTimeStr := c.QueryParam("end_time")

	// Validate parameters
	if userID == "" {
		return utils.BadRequestResponse(c, "User ID is required")
	}

	if role != "passenger" && role != "driver" {
		return utils.BadRequestResponse(c, "Role must be 'passenger' or 'driver'")
	}

	// Parse start time (default to 30 days ago if not provided)
	var startTime time.Time
	if startTimeStr == "" {
		startTime = time.Now().AddDate(0, -1, 0) // 1 month ago
	} else {
		var err error
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return utils.BadRequestResponse(c, "Invalid start time format, use RFC3339")
		}
	}

	// Parse end time (default to now if not provided)
	var endTime time.Time
	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		var err error
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return utils.BadRequestResponse(c, "Invalid end time format, use RFC3339")
		}
	}

	// Get ride history
	trips, err := h.RideUC.GetRideHistory(c.Request().Context(), userID, role, startTime, endTime)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride history retrieved successfully", trips)
}

// RateRide handles ride rating
func (h *RideHandler) RateRide(c echo.Context) error {
	// Get trip ID from path parameter
	tripID := c.Param("id")
	if tripID == "" {
		return utils.BadRequestResponse(c, "Trip ID is required")
	}

	// Parse request body
	type RateRideBody struct {
		UserID string  `json:"user_id"`
		Role   string  `json:"role"`
		Rating float64 `json:"rating"`
	}

	var req RateRideBody
	if err := c.Bind(&req); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate request
	if req.UserID == "" {
		return utils.BadRequestResponse(c, "User ID is required")
	}

	if req.Role != "passenger" && req.Role != "driver" {
		return utils.BadRequestResponse(c, "Role must be 'passenger' or 'driver'")
	}

	if req.Rating < 1 || req.Rating > 5 {
		return utils.BadRequestResponse(c, "Rating must be between 1 and 5")
	}

	// Rate ride
	if err := h.RideUC.RateRide(c.Request().Context(), tripID, req.UserID, req.Role, req.Rating); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Ride rated successfully", nil)
}
