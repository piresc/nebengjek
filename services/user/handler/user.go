package handler

import (
	"net/http"
	"strconv"

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

// RegisterDriver handles driver registration requests
func (h *UserHandler) RegisterDriver(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	if err := h.userUC.RegisterDriver(c.Request().Context(), &user); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusCreated, "Driver registered successfully", user)
}
