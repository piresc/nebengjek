package handler

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/services/gateway"
	gatewaywebsocket "github.com/piresc/nebengjek/services/gateway/handler/websocket"
	"github.com/piresc/nebengjek/services/gateway/repository"
)

// Handler coordinates all protocol handlers for the gateway service
type Handler struct {
	gatewayUC    gateway.GatewayUC
	cfg          *core.Config
	nrApp        *newrelic.Application
	wsHandler    *gatewaywebsocket.EchoWebSocketHandler
	proxyHandler *ProxyHandler
	sessionRepo  *repository.WSSessionRepository
	serverID     string
}

// NewHandler creates and initializes all handlers following clean architecture
func NewHandler(gatewayUC gateway.GatewayUC, cfg *core.Config, nrApp *newrelic.Application, wsHandler *gatewaywebsocket.EchoWebSocketHandler, sessionRepo *repository.WSSessionRepository, serverID string) *Handler {
	// Create proxy handler with UseCase dependency injection
	proxyHandler := NewProxyHandler(gatewayUC)

	return &Handler{
		gatewayUC:    gatewayUC,
		cfg:          cfg,
		nrApp:        nrApp,
		wsHandler:    wsHandler,
		proxyHandler: proxyHandler,
		sessionRepo:  sessionRepo,
		serverID:     serverID,
	}
}


// GetWebSocketJWTMiddleware returns a custom JWT middleware for WebSocket requests
func (h *Handler) GetWebSocketJWTMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Parse JWT token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(401, "Missing authorization header")
			}

			if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				return echo.NewHTTPError(401, "Invalid authorization header format")
			}

			tokenString := authHeader[7:]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(401, "Invalid signing method")
				}
				return []byte(h.cfg.JWT.Secret), nil
			})

			if err != nil || !token.Valid {
				return echo.NewHTTPError(401, "Invalid token")
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if userID, exists := claims["user_id"]; exists {
					c.Set("user_id", userID)
				}
				if role, exists := claims["role"]; exists {
					c.Set("role", role)
				}
				return next(c)
			}

			return echo.NewHTTPError(401, "Invalid token claims")
		}
	}
}
