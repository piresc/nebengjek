package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/services/user"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userUC user.UserUC
}

// NewUserHandler creates a new user handler
func NewUserHandler(userUC user.UserUC) *UserHandler {
	return &UserHandler{userUC: userUC}
}

// RegisterRoutes registers the user API routes
func (h *UserHandler) RegisterRoutes(e *echo.Echo) {
	// Public routes (JWT authentication)
	e.POST("/auth/login", h.GenerateOTP) // Generates OTP via SMS
	e.POST("/auth/verify", h.VerifyOTP)  // Validates OTP and issues JWT

	// Service-to-service routes (API key authentication)
	serviceRoutes := e.Group("/internal")
	serviceRoutes.Use(
		middleware.ValidateAPIKey(
			"match-service",
			"billing-service",
			"trip-service"))
	serviceRoutes.GET("/users/:id", h.GetUser)
	serviceRoutes.GET("/users", h.ListUsers)
	serviceRoutes.POST("/drivers", h.RegisterDriver)

	// User routes (JWT authentication)
	e.GET("/users/:id", h.GetUser)
	e.PUT("/users/:id", h.UpdateUser)
	e.DELETE("/users/:id", h.DeactivateUser)

	// Beacon routes (JWT authentication)
	e.POST("/beacon", h.ToggleBeacon)
}
