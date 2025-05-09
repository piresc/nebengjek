package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
	"github.com/piresc/nebengjek/services/users"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	userUC users.UserUC
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(userUC users.UserUC) *AuthHandler {
	return &AuthHandler{
		userUC: userUC,
	}
}

// GenerateOTP handles OTP generation requests via SMS
func (h *AuthHandler) GenerateOTP(c echo.Context) error {
	var request models.LoginRequest
	if err := c.Bind(&request); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate MSISDN
	if request.MSISDN == "" {
		return utils.BadRequestResponse(c, "MSISDN is required")
	}

	// Generate and send OTP via SMS
	if err := h.userUC.GenerateOTP(c.Request().Context(), request.MSISDN); err != nil {
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, err.Error())
	}

	return utils.SuccessResponse(c, http.StatusOK, "OTP sent successfully", nil)
}

// VerifyOTP handles OTP verification requests
func (h *AuthHandler) VerifyOTP(c echo.Context) error {
	var request models.VerifyRequest
	if err := c.Bind(&request); err != nil {
		return utils.BadRequestResponse(c, "Invalid request payload")
	}

	// Validate request
	if request.MSISDN == "" || request.OTP == "" {
		return utils.BadRequestResponse(c, "MSISDN and OTP are required")
	}

	// Verify OTP and generate JWT token
	response, err := h.userUC.VerifyOTP(c.Request().Context(), request.MSISDN, request.OTP)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Invalid OTP")
	}

	return utils.SuccessResponse(c, http.StatusOK, "OTP verified successfully", response)
}
