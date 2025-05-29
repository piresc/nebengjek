package utils

import (
	"encoding/json"
	"fmt"
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

// ParseJSONResponse reads a JSON response from an HTTP response body and extracts the data
// into the provided target interface. This function is designed to work with the standard
// Response structure used throughout the application.
//
// Example usage:
//
//	var user models.User
//	if err := utils.ParseJSONResponse(resp.Body, &user); err != nil {
//	    return nil, err
//	}
func ParseJSONResponse(respBody []byte, target interface{}) error {
	var response Response
	if err := json.Unmarshal(respBody, &response); err != nil {
		return fmt.Errorf("failed to decode API response: %w, body: %s", err, string(respBody))
	}

	// Check if there was an error in the response
	if !response.Success {
		return fmt.Errorf("API error: %s", response.Error)
	}

	// If there's no data, just return
	if response.Data == nil {
		return nil
	}

	// Check if data is another Response object (handle nested responses)
	dataMap, ok := response.Data.(map[string]interface{})
	if ok {
		// Check if this looks like a nested Response structure
		if _, hasSuccess := dataMap["success"]; hasSuccess {
			nestedSuccess, _ := dataMap["success"].(bool)
			nestedError, hasError := dataMap["error"].(string)

			if hasError && !nestedSuccess {
				return fmt.Errorf("API error in nested response: %s", nestedError)
			}

			// If there's a nested data field, use that instead
			if nestedData, hasData := dataMap["data"]; hasData && nestedData != nil {
				nestedDataJSON, err := json.Marshal(nestedData)
				if err != nil {
					return fmt.Errorf("failed to re-marshal nested response data: %w", err)
				}

				if err := json.Unmarshal(nestedDataJSON, target); err != nil {
					return fmt.Errorf("failed to unmarshal nested response data into target: %w", err)
				}

				return nil
			}
		}
	}

	// Standard case: marshal the data field to JSON
	dataJSON, err := json.Marshal(response.Data)
	if err != nil {
		return fmt.Errorf("failed to re-marshal response data: %w", err)
	}

	// Unmarshal into the target
	if err := json.Unmarshal(dataJSON, target); err != nil {
		return fmt.Errorf("failed to unmarshal response data into target: %w", err)
	}

	return nil
}
