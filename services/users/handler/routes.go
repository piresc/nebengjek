package handler

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/handler/http"
	"github.com/piresc/nebengjek/services/users/handler/nats"
	"github.com/piresc/nebengjek/services/users/handler/websocket"
)

// Handler coordinates all protocol handlers for the user service
type Handler struct {
	userHandler   *http.UserHandler
	authHandler   *http.AuthHandler
	echoWSHandler *websocket.EchoWebSocketHandler
	natsHandler   *nats.NatsHandler
	cfg           *models.Config
}

// NewHandler creates and initializes all handlers
func NewHandler(
	userHandler *http.UserHandler,
	authHandler *http.AuthHandler,
	echoWSHandler *websocket.EchoWebSocketHandler,
	natsHandler *nats.NatsHandler,
	cfg *models.Config,
) *Handler {

	return &Handler{
		userHandler:   userHandler,
		authHandler:   authHandler,
		echoWSHandler: echoWSHandler,
		natsHandler:   natsHandler,
		cfg:           cfg,
	}
}

// GetJWTMiddleware returns the configured JWT middleware for HTTP requests
func (h *Handler) GetJWTMiddleware() echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(h.cfg.JWT.Secret),
		SuccessHandler: func(c echo.Context) {
			// Parse the token directly from Authorization header to avoid type conflicts
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenString := authHeader[7:]
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return []byte(h.cfg.JWT.Secret), nil
				})
				if err == nil && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						if userID, exists := claims["user_id"]; exists {
							c.Set("user_id", userID)
						}
						if role, exists := claims["role"]; exists {
							c.Set("role", role)
						}
					}
				}
			}
		},
	})
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
				return []byte(h.cfg.JWT.Secret), nil
			})

			if err != nil || !token.Valid {
				return echo.NewHTTPError(401, "Invalid token")
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if userID, exists := claims["user_id"]; exists {
					// Validate that userID is not empty or nil
					userIDStr := fmt.Sprintf("%v", userID)
					if userIDStr == "" || userIDStr == "<nil>" || userIDStr == "00000000-0000-0000-0000-000000000000" {
						return echo.NewHTTPError(401, "Invalid user ID in token")
					}
					c.Set("user_id", userID)
				} else {
					return echo.NewHTTPError(401, "Missing user ID in token")
				}
				if role, exists := claims["role"]; exists {
					c.Set("role", role)
				} else {
					return echo.NewHTTPError(401, "Missing role in token")
				}
				return next(c)
			}

			return echo.NewHTTPError(401, "Invalid token claims")
		}
	}
}

// RegisterRoutes registers all protocol handlers and their routes
func (h *Handler) RegisterRoutes(e *echo.Echo, unifiedMiddleware *middleware.UnifiedMiddleware) {
	// Public routes (no authentication required)
	authGroup := e.Group("/auth")
	authGroup.POST("/otp/generate", h.authHandler.GenerateOTP)
	authGroup.POST("/otp/verify", h.authHandler.VerifyOTP)

	// Protected routes with JWT middleware (user-facing)
	protected := e.Group("", h.GetJWTMiddleware())

	// User routes
	userGroup := protected.Group("/users")
	userGroup.POST("", h.userHandler.CreateUser)
	userGroup.GET("/:id", h.userHandler.GetUser)

	// Driver routes
	driverGroup := protected.Group("/drivers")
	driverGroup.POST("/register", h.userHandler.RegisterDriver)

	// WebSocket routes - use custom WebSocket JWT middleware
	wsGroup := e.Group("/ws", h.GetWebSocketJWTMiddleware())
	wsGroup.GET("", h.echoWSHandler.HandleWebSocket)

}
