package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// CreateUser handles user creation requests
func (h *UserHandler) CreateUser(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	if err := h.userUC.RegisterUser(c.Request().Context(), &user); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	// Clear password before responding
	user.Password = ""

	return utils.SuccessResponse(c, http.StatusCreated, "User created successfully", user)
}

// GetUser handles user retrieval requests
func (h *UserHandler) GetUser(c echo.Context) error {
	id := c.Param("id")

	user, err := h.userUC.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusNotFound, "User not found")
	}

	return utils.SuccessResponse(c, http.StatusOK, "User retrieved successfully", user)
}

// UpdateUser handles user update requests
func (h *UserHandler) UpdateUser(c echo.Context) error {
	id := c.Param("id")

	var user models.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Set ID from URL
	user.ID = id

	if err := h.userUC.UpdateUserProfile(c.Request().Context(), &user); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "User updated successfully", nil)
}

// DeactivateUser handles user deactivation requests
func (h *UserHandler) DeactivateUser(c echo.Context) error {
	id := c.Param("id")

	if err := h.userUC.DeactivateUser(c.Request().Context(), id); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "User deactivated successfully", nil)
}

// ListUsers handles user listing requests
func (h *UserHandler) ListUsers(c echo.Context) error {
	// Parse pagination parameters
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	// Set default limit if not provided
	if limit <= 0 {
		limit = 10
	}

	users, err := h.userUC.ListUsers(c.Request().Context(), offset, limit)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", users)
}

// Login handles user authentication requests
func (h *UserHandler) Login(c echo.Context) error {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&credentials); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	user, err := h.userUC.AuthenticateUser(c.Request().Context(), credentials.Email, credentials.Password)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Invalid credentials")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Login successful", user)
}

// RegisterDriver handles driver registration requests
func (h *UserHandler) RegisterDriver(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	if err := h.userUC.RegisterDriver(c.Request().Context(), &user); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	// Clear password before responding
	user.Password = ""

	return utils.SuccessResponse(c, http.StatusCreated, "Driver registered successfully", user)
}

// UpdateDriverLocation handles driver location update requests
func (h *UserHandler) UpdateDriverLocation(c echo.Context) error {
	id := c.Param("id")

	var location models.Location
	if err := c.Bind(&location); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	if err := h.userUC.UpdateDriverLocation(c.Request().Context(), id, &location); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Driver location updated successfully", nil)
}

// UpdateDriverAvailability handles driver availability update requests
func (h *UserHandler) UpdateDriverAvailability(c echo.Context) error {
	id := c.Param("id")

	var availability struct {
		IsAvailable bool `json:"is_available"`
	}

	if err := c.Bind(&availability); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	if err := h.userUC.UpdateDriverAvailability(c.Request().Context(), id, availability.IsAvailable); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Driver availability updated successfully", nil)
}

// GetNearbyDrivers handles nearby drivers retrieval requests
func (h *UserHandler) GetNearbyDrivers(c echo.Context) error {
	// Parse location parameters
	lat, err := strconv.ParseFloat(c.QueryParam("latitude"), 64)
	if err != nil {
		return utils.BadRequestResponse(c, "Invalid latitude")
	}

	lon, err := strconv.ParseFloat(c.QueryParam("longitude"), 64)
	if err != nil {
		return utils.BadRequestResponse(c, "Invalid longitude")
	}

	radius, err := strconv.ParseFloat(c.QueryParam("radius"), 64)
	if err != nil || radius <= 0 {
		radius = 5.0 // Default radius in kilometers
	}

	location := &models.Location{
		Latitude:  lat,
		Longitude: lon,
		Timestamp: time.Now(),
	}

	drivers, err := h.userUC.GetNearbyDrivers(c.Request().Context(), location, radius)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Nearby drivers retrieved successfully", drivers)
}

// VerifyDriver handles driver verification requests
func (h *UserHandler) VerifyDriver(c echo.Context) error {
	id := c.Param("id")

	if err := h.userUC.VerifyDriver(c.Request().Context(), id); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "Driver verified successfully", nil)
}

// Note: Helper functions for HTTP responses are no longer needed as we're using Echo's built-in response utilities
