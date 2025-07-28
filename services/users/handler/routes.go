package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/handler/http"
)

// Handler coordinates all protocol handlers for the user service
type Handler struct {
	userHandler *http.UserHandler
	authHandler *http.AuthHandler
	cfg         *models.Config
}

// NewHandler creates and initializes all handlers
func NewHandler(
	userHandler *http.UserHandler,
	authHandler *http.AuthHandler,
	cfg *models.Config,
) *Handler {

	return &Handler{
		userHandler: userHandler,
		authHandler: authHandler,
		cfg:         cfg,
	}
}

// RegisterRoutes registers all protocol handlers and their routes
func (h *Handler) RegisterRoutes(e *echo.Echo, mw *middleware.Middleware) {
	internal := e.Group("/internal", mw.APIKeyHandler("users-service"))

	// Authentication routes (called by Gateway)
	authGroup := internal.Group("/auth")
	authGroup.POST("/otp/generate", h.authHandler.GenerateOTP)
	authGroup.POST("/otp/verify", h.authHandler.VerifyOTP)

	// User routes (called by Gateway)
	userGroup := internal.Group("/users")
	userGroup.POST("", h.userHandler.CreateUser)
	userGroup.GET("/:id", h.userHandler.GetUser)

	// Driver routes (called by Gateway)
	driverGroup := internal.Group("/drivers")
	driverGroup.POST("/register", h.userHandler.RegisterDriver)

	// Finder routes (called by Gateway)
	finderGroup := internal.Group("/finder")
	finderGroup.POST("/update", h.userHandler.UpdateFinderStatus)

	// Beacon routes (called by Gateway)
	beaconGroup := internal.Group("/beacon")
	beaconGroup.POST("/update", h.userHandler.UpdateBeaconStatus)

}
