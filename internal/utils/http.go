package utils

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
}

// SuccessResponse sends a success response with data
func SuccessResponse(c echo.Context, statusCode int, message string, data interface{}) error {
	return c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorResponseHandler sends an error response
func ErrorResponseHandler(c echo.Context, statusCode int, errorMessage string) error {
	return c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error:   errorMessage,
		Code:    statusCode,
	})
}

// BadRequestResponse sends a 400 Bad Request response
func BadRequestResponse(c echo.Context, errorMessage string) error {
	return ErrorResponseHandler(c, http.StatusBadRequest, errorMessage)
}

// UnauthorizedResponse sends a 401 Unauthorized response
func UnauthorizedResponse(c echo.Context, errorMessage string) error {
	if errorMessage == "" {
		errorMessage = "Unauthorized"
	}
	return ErrorResponseHandler(c, http.StatusUnauthorized, errorMessage)
}

// ForbiddenResponse sends a 403 Forbidden response
func ForbiddenResponse(c echo.Context, errorMessage string) error {
	if errorMessage == "" {
		errorMessage = "Forbidden"
	}
	return ErrorResponseHandler(c, http.StatusForbidden, errorMessage)
}

// NotFoundResponse sends a 404 Not Found response
func NotFoundResponse(c echo.Context, errorMessage string) error {
	if errorMessage == "" {
		errorMessage = "Resource not found"
	}
	return ErrorResponseHandler(c, http.StatusNotFound, errorMessage)
}

// InternalServerErrorResponse sends a 500 Internal Server Error response
func InternalServerErrorResponse(c echo.Context, errorMessage string) error {
	if errorMessage == "" {
		errorMessage = "Internal server error"
	}
	return ErrorResponseHandler(c, http.StatusInternalServerError, errorMessage)
}

// ServiceUnavailableResponse sends a 503 Service Unavailable response
func ServiceUnavailableResponse(c echo.Context, errorMessage string) error {
	if errorMessage == "" {
		errorMessage = "Service unavailable"
	}
	return ErrorResponseHandler(c, http.StatusServiceUnavailable, errorMessage)
}
