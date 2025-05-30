package handler

import (
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/handler/http"
	"github.com/piresc/nebengjek/services/users/handler/nats"
	"github.com/piresc/nebengjek/services/users/handler/websocket"
)

// Handler coordinates all protocol handlers for the user service
type Handler struct {
	userHandler *http.UserHandler
	authHandler *http.AuthHandler
	wsManager   *websocket.WebSocketManager
	natsHandler *nats.NatsHandler
	cfg         *models.Config
}

// NewHandler creates and initializes all handlers
func NewHandler(
	userHandler *http.UserHandler,
	authHandler *http.AuthHandler,
	wsManager *websocket.WebSocketManager,
	natsHandler *nats.NatsHandler,
	cfg *models.Config,
) *Handler {

	return &Handler{
		userHandler: userHandler,
		authHandler: authHandler,
		wsManager:   wsManager,
		natsHandler: natsHandler,
		cfg:         cfg,
	}
}

// GetJWTMiddleware returns the configured JWT middleware
func (h *Handler) GetJWTMiddleware() echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(h.cfg.JWT.Secret),
	})
}

// RegisterRoutes registers all protocol handlers and their routes
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Auth routes (public)
	authGroup := e.Group("/auth")
	authGroup.POST("/otp/generate", h.authHandler.GenerateOTP)
	authGroup.POST("/otp/verify", h.authHandler.VerifyOTP)

	// Protected routes with JWT middleware
	protected := e.Group("", h.GetJWTMiddleware())

	// User routes
	userGroup := protected.Group("/users")
	userGroup.POST("", h.userHandler.CreateUser)
	userGroup.GET("/:id", h.userHandler.GetUser)

	// Driver routes
	driverGroup := protected.Group("/drivers")
	driverGroup.POST("/register", h.userHandler.RegisterDriver)

	// WebSocket routes
	wsGroup := protected.Group("/ws")
	wsGroup.GET("", h.wsManager.HandleWebSocket)
}
