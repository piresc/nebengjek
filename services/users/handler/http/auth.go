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

	// Additional validation for MSISDN format
	if len(request.MSISDN) < 10 || len(request.MSISDN) > 15 {
		return utils.BadRequestResponse(c, "Invalid MSISDN format")
	}

	// Generate and send OTP via SMS
	if err := h.userUC.GenerateOTP(c.Request().Context(), request.MSISDN); err != nil {
		if err.Error() == "invalid MSISDN format or not a Telkomsel number" {
			return utils.BadRequestResponse(c, "Invalid Telkomsel number")
		}
		return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Failed to generate OTP")
	}

	return utils.SuccessResponse(c, http.StatusOK, "OTP sent successfully", map[string]interface{}{
		"msisdn": request.MSISDN,
		"expires_in": 300, // 5 minutes
	})
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

	// Validate OTP format (should be 4 digits)
	if len(request.OTP) != 4 {
		return utils.BadRequestResponse(c, "OTP must be 4 digits")
	}

	// Verify OTP and generate JWT token
	response, err := h.userUC.VerifyOTP(c.Request().Context(), request.MSISDN, request.OTP)
	if err != nil {
		if err.Error() == "invalid MSISDN format or not a Telkomsel number" {
			return utils.BadRequestResponse(c, "Invalid Telkomsel number")
		}
		if err.Error() == "OTP not found or expired" {
			return utils.UnauthorizedResponse(c, "OTP expired or not found")
		}
		return utils.UnauthorizedResponse(c, "Invalid OTP")
	}

	return utils.SuccessResponse(c, http.StatusOK, "Authentication successful", response)
}
