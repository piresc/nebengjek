package handler

import (
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/user"
	"github.com/piresc/nebengjek/services/user/handler/http"
	"github.com/piresc/nebengjek/services/user/handler/nats"
	"github.com/piresc/nebengjek/services/user/handler/websocket"
)

// Handler coordinates all protocol handlers for the user service
type Handler struct {
	userHandler *http.UserHandler
	authHandler *http.AuthHandler
	wsManager   *websocket.WebSocketManager
	natsHandler *nats.Handler
	jwtConfig   models.JWTConfig
}

// NewHandler creates and initializes all handlers
func NewUserHandler(userUC user.UserUC, natsURL string, jwtConfig models.JWTConfig) (*Handler, error) {
	// Initialize WebSocket manager
	wsManager := websocket.NewWebSocketManager(userUC, jwtConfig)

	// Initialize NATS handler
	natsHandler, err := nats.NewHandler(wsManager, natsURL)
	if err != nil {
		return nil, err
	}

	return &Handler{
		userHandler: http.NewUserHandler(userUC),
		authHandler: http.NewAuthHandler(userUC),
		wsManager:   wsManager,
		natsHandler: natsHandler,
		jwtConfig:   jwtConfig,
	}, nil
}

// RegisterRoutes registers all protocol handlers and their routes
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// Auth routes (public)
	authGroup := e.Group("/auth")
	authGroup.POST("/otp/generate", h.authHandler.GenerateOTP)
	authGroup.POST("/otp/verify", h.authHandler.VerifyOTP)

	// Configure JWT middleware
	config := echojwt.Config{
		SigningKey: []byte(h.jwtConfig.Secret),
	}

	// Protected routes
	protected := e.Group("", echojwt.WithConfig(config))

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
