package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/models/user"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/users"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userUC users.UserUC
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userUC users.UserUC,
) *UserHandler {
	return &UserHandler{
		userUC: userUC,
	}
}

// CreateUser handles user creation requests
func (h *UserHandler) CreateUser(c echo.Context) error {
	var user user.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	err := h.userUC.RegisterUser(c.Request().Context(), &user)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to create user")
	}

	return utils.SuccessResponse(c, http.StatusCreated, "User created successfully", user)
}

// GetUser handles user retrieval requests
func (h *UserHandler) GetUser(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return utils.BadRequestResponse(c, "Invalid user ID")
	}

	user, err := h.userUC.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to retrieve user")
	}

	return utils.SuccessResponse(c, http.StatusOK, "User retrieved successfully", user)
}

// RegisterDriver handles driver registration requests
func (h *UserHandler) RegisterDriver(c echo.Context) error {
	var user user.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	err := h.userUC.RegisterDriver(c.Request().Context(), &user)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to register driver")
	}

	return utils.SuccessResponse(c, http.StatusCreated, "Driver registered successfully", user)
}

// UpdateFinderStatus handles finder status update requests
func (h *UserHandler) UpdateFinderStatus(c echo.Context) error {
	var finderReq core.FinderRequest
	if err := c.Bind(&finderReq); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	err := h.userUC.UpdateFinderStatus(c.Request().Context(), &finderReq)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to update finder status")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Finder status updated successfully", nil)
}

// UpdateBeaconStatus handles beacon status update requests
func (h *UserHandler) UpdateBeaconStatus(c echo.Context) error {
	var beaconReq core.BeaconRequest
	if err := c.Bind(&beaconReq); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	err := h.userUC.UpdateBeaconStatus(c.Request().Context(), &beaconReq)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to update beacon status")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Beacon status updated successfully", nil)
}
