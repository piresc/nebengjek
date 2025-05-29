package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}

// Success response structure
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// BadRequestResponse returns a standard bad request response
func BadRequestResponse(c echo.Context, message string) error {
	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: message,
	})
}

// ErrorResponseHandler returns a standard error response
func ErrorResponseHandler(c echo.Context, status int, message string) error {
	return c.JSON(status, ErrorResponse{
		Error: message,
	})
}

// SuccessResponseWithData returns a standard success response with data
func SuccessResponseWithData(c echo.Context, status int, message string, data interface{}) error {
	return c.JSON(status, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}
