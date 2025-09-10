package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/gateway/mocks"
	gatewaywebsocket "github.com/piresc/nebengjek/services/gateway/handler/websocket"
	"github.com/piresc/nebengjek/services/gateway/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret: "test-secret",
		},
	}
	nrApp := &newrelic.Application{}
	wsHandler := &gatewaywebsocket.EchoWebSocketHandler{}
	sessionRepo := &repository.WSSessionRepository{}
	serverID := "test-server"

	handler := NewHandler(mockGatewayUC, cfg, nrApp, wsHandler, sessionRepo, serverID)

	assert.NotNil(t, handler)
	assert.Equal(t, mockGatewayUC, handler.gatewayUC)
	assert.Equal(t, cfg, handler.cfg)
	assert.Equal(t, nrApp, handler.nrApp)
	assert.Equal(t, wsHandler, handler.wsHandler)
	assert.Equal(t, sessionRepo, handler.sessionRepo)
	assert.Equal(t, serverID, handler.serverID)
	assert.NotNil(t, handler.proxyHandler)
}

func TestHandler_GetWebSocketJWTMiddleware(t *testing.T) {
	tests := []struct {
		name               string
		authHeader         string
		jwtSecret          string
		expectedStatusCode int
		expectedError      string
		setupToken         func(secret string) string
	}{
		{
			name:               "missing authorization header",
			authHeader:         "",
			jwtSecret:          "test-secret",
			expectedStatusCode: 401,
			expectedError:      "Missing authorization header",
		},
		{
			name:               "invalid header format - no Bearer",
			authHeader:         "InvalidToken",
			jwtSecret:          "test-secret",
			expectedStatusCode: 401,
			expectedError:      "Invalid authorization header format",
		},
		{
			name:               "invalid header format - too short",
			authHeader:         "Bear",
			jwtSecret:          "test-secret",
			expectedStatusCode: 401,
			expectedError:      "Invalid authorization header format",
		},
		{
			name:               "invalid token",
			authHeader:         "Bearer invalid.token.here",
			jwtSecret:          "test-secret",
			expectedStatusCode: 401,
			expectedError:      "Invalid token",
		},
		{
			name:      "valid token",
			jwtSecret: "test-secret",
			setupToken: func(secret string) string {
				claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": "user-123",
					"role":    "driver",
					"exp":     jwt.NewNumericDate(jwt.TimeFunc().Add(24 * 60 * 60 * 1000000000)), // 24 hours
				})
				tokenString, _ := claims.SignedString([]byte(secret))
				return "Bearer " + tokenString
			},
			expectedStatusCode: 200,
		},
		{
			name:      "valid token without user_id",
			jwtSecret: "test-secret",
			setupToken: func(secret string) string {
				claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"role": "driver",
					"exp":  jwt.NewNumericDate(jwt.TimeFunc().Add(24 * 60 * 60 * 1000000000)),
				})
				tokenString, _ := claims.SignedString([]byte(secret))
				return "Bearer " + tokenString
			},
			expectedStatusCode: 200,
		},
		{
			name:      "token with wrong signing method",
			jwtSecret: "test-secret",
			setupToken: func(secret string) string {
				// This will create a token with RS256 instead of expected HS256
				// This will fail to sign properly, but we return a malformed token
				return "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature"
			},
			expectedStatusCode: 401,
			expectedError:      "Invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
			cfg := &models.Config{
				JWT: models.JWTConfig{
					Secret: tt.jwtSecret,
				},
			}
			nrApp := &newrelic.Application{}
			wsHandler := &gatewaywebsocket.EchoWebSocketHandler{}

			handler := NewHandler(mockGatewayUC, cfg, nrApp, wsHandler, nil, "test-server")

			// Create a dummy next handler
			nextCalled := false
			next := func(c echo.Context) error {
				nextCalled = true
				return c.String(200, "OK")
			}

			middleware := handler.GetWebSocketJWTMiddleware()
			middlewareHandler := middleware(next)

			// Set up the auth header
			authHeader := tt.authHeader
			if tt.setupToken != nil {
				authHeader = tt.setupToken(tt.jwtSecret)
			}

			req := httptest.NewRequest("GET", "/ws", nil)
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			err := middlewareHandler(c)

			if tt.expectedStatusCode == 200 {
				assert.NoError(t, err)
				assert.True(t, nextCalled)
				assert.Equal(t, "OK", rec.Body.String())
			} else {
				assert.Error(t, err)
				assert.False(t, nextCalled)
				if httpErr, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatusCode, httpErr.Code)
					if tt.expectedError != "" {
						assert.Contains(t, httpErr.Message, tt.expectedError)
					}
				}
			}
		})
	}
}

func TestHandler_GetWebSocketJWTMiddleware_SuccessfulTokenParsing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret: "test-secret",
		},
	}
	nrApp := &newrelic.Application{}
	wsHandler := &gatewaywebsocket.EchoWebSocketHandler{}

	handler := NewHandler(mockGatewayUC, cfg, nrApp, wsHandler, nil, "test-server")

	// Create a valid JWT token
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user-123",
		"role":    "driver",
		"exp":     jwt.NewNumericDate(jwt.TimeFunc().Add(24 * 60 * 60 * 1000000000)),
	})
	tokenString, err := claims.SignedString([]byte("test-secret"))
	assert.NoError(t, err)

	nextCalled := false
	var capturedContext echo.Context
	next := func(c echo.Context) error {
		nextCalled = true
		capturedContext = c
		return c.String(200, "OK")
	}

	middleware := handler.GetWebSocketJWTMiddleware()
	middlewareHandler := middleware(next)

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err = middlewareHandler(c)

	assert.NoError(t, err)
	assert.True(t, nextCalled)

	// Verify that user context was set correctly
	assert.Equal(t, "user-123", capturedContext.Get("user_id"))
	assert.Equal(t, "driver", capturedContext.Get("role"))
}

func TestHandler_GetWebSocketJWTMiddleware_InvalidClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret: "test-secret",
		},
	}
	nrApp := &newrelic.Application{}
	wsHandler := &gatewaywebsocket.EchoWebSocketHandler{}

	handler := NewHandler(mockGatewayUC, cfg, nrApp, wsHandler, nil, "test-server")

	// Create a token with valid structure but empty MapClaims (this should succeed)
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	tokenString, err := claims.SignedString([]byte("test-secret"))
	assert.NoError(t, err)

	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.String(200, "OK")
	}

	middleware := handler.GetWebSocketJWTMiddleware()
	middlewareHandler := middleware(next)

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err = middlewareHandler(c)

	// Empty MapClaims are valid, so this should succeed
	assert.NoError(t, err)
	assert.True(t, nextCalled)
}