package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSuccessResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		data       interface{}
	}{
		{
			name:       "Success with string data",
			statusCode: http.StatusOK,
			message:    "Operation successful",
			data:       "test data",
		},
		{
			name:       "Success with map data",
			statusCode: http.StatusCreated,
			message:    "Resource created",
			data:       map[string]interface{}{"id": "123", "name": "test"},
		},
		{
			name:       "Success with nil data",
			statusCode: http.StatusOK,
			message:    "Success",
			data:       nil,
		},
		{
			name:       "Success with empty message",
			statusCode: http.StatusOK,
			message:    "",
			data:       "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := SuccessResponse(c, tt.statusCode, tt.message, tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.statusCode, rec.Code)

			var response Response
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response.Success)
			assert.Equal(t, tt.message, response.Message)
			assert.Equal(t, tt.data, response.Data)
		})
	}
}

func TestErrorResponseHandler(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		errorMessage string
	}{
		{
			name:         "Internal server error",
			statusCode:   http.StatusInternalServerError,
			errorMessage: "Internal server error occurred",
		},
		{
			name:         "Bad request",
			statusCode:   http.StatusBadRequest,
			errorMessage: "Invalid request",
		},
		{
			name:         "Empty error message",
			statusCode:   http.StatusNotFound,
			errorMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := ErrorResponseHandler(c, tt.statusCode, tt.errorMessage)
			assert.NoError(t, err)
			assert.Equal(t, tt.statusCode, rec.Code)

			var response ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.errorMessage, response.Error)
			assert.Equal(t, tt.statusCode, response.Code)
		})
	}
}

func TestBadRequestResponse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	errorMessage := "Invalid input"
	err := BadRequestResponse(c, errorMessage)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, errorMessage, response.Error)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestUnauthorizedResponse(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expected     string
	}{
		{
			name:         "Custom error message",
			errorMessage: "Invalid token",
			expected:     "Invalid token",
		},
		{
			name:         "Empty error message",
			errorMessage: "",
			expected:     "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := UnauthorizedResponse(c, tt.errorMessage)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)

			var response ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expected, response.Error)
			assert.Equal(t, http.StatusUnauthorized, response.Code)
		})
	}
}

func TestForbiddenResponse(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expected     string
	}{
		{
			name:         "Custom error message",
			errorMessage: "Access denied",
			expected:     "Access denied",
		},
		{
			name:         "Empty error message",
			errorMessage: "",
			expected:     "Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := ForbiddenResponse(c, tt.errorMessage)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusForbidden, rec.Code)

			var response ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expected, response.Error)
			assert.Equal(t, http.StatusForbidden, response.Code)
		})
	}
}

func TestNotFoundResponse(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expected     string
	}{
		{
			name:         "Custom error message",
			errorMessage: "User not found",
			expected:     "User not found",
		},
		{
			name:         "Empty error message",
			errorMessage: "",
			expected:     "Resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := NotFoundResponse(c, tt.errorMessage)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, rec.Code)

			var response ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expected, response.Error)
			assert.Equal(t, http.StatusNotFound, response.Code)
		})
	}
}

func TestInternalServerErrorResponse(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expected     string
	}{
		{
			name:         "Custom error message",
			errorMessage: "Database connection failed",
			expected:     "Database connection failed",
		},
		{
			name:         "Empty error message",
			errorMessage: "",
			expected:     "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := InternalServerErrorResponse(c, tt.errorMessage)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			var response ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expected, response.Error)
			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	}
}

func TestServiceUnavailableResponse(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expected     string
	}{
		{
			name:         "Custom error message",
			errorMessage: "Service temporarily down",
			expected:     "Service temporarily down",
		},
		{
			name:         "Empty error message",
			errorMessage: "",
			expected:     "Service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := ServiceUnavailableResponse(c, tt.errorMessage)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

			var response ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response.Success)
			assert.Equal(t, tt.expected, response.Error)
			assert.Equal(t, http.StatusServiceUnavailable, response.Code)
		})
	}
}

func TestParseJSONResponse(t *testing.T) {
	tests := []struct {
		name        string
		respBody    []byte
		target      interface{}
		expectError bool
		expected    interface{}
	}{
		{
			name: "Valid response with string data",
			respBody: []byte(`{
				"success": true,
				"message": "Success",
				"data": "test string"
			}`),
			target:      new(string),
			expectError: false,
			expected:    "test string",
		},
		{
			name: "Valid response with map data",
			respBody: []byte(`{
				"success": true,
				"message": "Success",
				"data": {"id": "123", "name": "test"}
			}`),
			target:      new(map[string]interface{}),
			expectError: false,
			expected:    map[string]interface{}{"id": "123", "name": "test"},
		},
		{
			name: "Error response",
			respBody: []byte(`{
				"success": false,
				"error": "Something went wrong"
			}`),
			target:      new(string),
			expectError: true,
			expected:    nil,
		},
		{
			name: "Invalid JSON",
			respBody: []byte(`{invalid json}`),
			target:      new(string),
			expectError: true,
			expected:    nil,
		},
		{
			name: "Nil data",
			respBody: []byte(`{
				"success": true,
				"message": "Success",
				"data": null
			}`),
			target:      new(string),
			expectError: false,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseJSONResponse(tt.respBody, tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected != nil {
					// Dereference pointer to get actual value
					switch v := tt.target.(type) {
					case *string:
						assert.Equal(t, tt.expected, *v)
					case *map[string]interface{}:
						assert.Equal(t, tt.expected, *v)
					}
				}
			}
		})
	}
}

// Test Response and ErrorResponse structs
func TestResponseStructs(t *testing.T) {
	t.Run("Response struct", func(t *testing.T) {
		resp := Response{
			Success: true,
			Message: "Test message",
			Data:    "test data",
			Error:   "",
		}

		assert.True(t, resp.Success)
		assert.Equal(t, "Test message", resp.Message)
		assert.Equal(t, "test data", resp.Data)
		assert.Empty(t, resp.Error)
	})

	t.Run("ErrorResponse struct", func(t *testing.T) {
		resp := ErrorResponse{
			Success: false,
			Error:   "Test error",
			Code:    400,
		}

		assert.False(t, resp.Success)
		assert.Equal(t, "Test error", resp.Error)
		assert.Equal(t, 400, resp.Code)
	})
}