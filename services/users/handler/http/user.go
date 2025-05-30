package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
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
