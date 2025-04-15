package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/user"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userUC user.UserUC
}

// NewUserHandler creates a new user handler
func NewUserHandler(userUC user.UserUC) *UserHandler {
	return &UserHandler{
		userUC: userUC,
	}
}

// CreateUser handles user creation requests
func (h *UserHandler) CreateUser(c echo.Context) error {
	var user models.User
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

// UpdateUser handles user update requests
func (h *UserHandler) UpdateUser(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return utils.BadRequestResponse(c, "Invalid user ID")
	}

	var user models.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Set ID from URL parameter
	user.ID = userID

	err := h.userUC.UpdateUserProfile(c.Request().Context(), &user)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to update user")
	}

	return utils.SuccessResponse(c, http.StatusOK, "User updated successfully", user)
}

// DeactivateUser handles user deactivation requests
func (h *UserHandler) DeactivateUser(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return utils.BadRequestResponse(c, "Invalid user ID")
	}

	err := h.userUC.DeactivateUser(c.Request().Context(), userID)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to deactivate user")
	}

	return utils.SuccessResponse(c, http.StatusOK, "User deactivated successfully", nil)
}

// ListUsers handles user listing requests
func (h *UserHandler) ListUsers(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	users, err := h.userUC.ListUsers(c.Request().Context(), page, limit)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to list users")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", users)
}

// RegisterDriver handles driver registration requests
func (h *UserHandler) RegisterDriver(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	err := h.userUC.RegisterDriver(c.Request().Context(), &user)
	if err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to register driver")
	}

	return utils.SuccessResponse(c, http.StatusCreated, "Driver registered successfully", user)
}
