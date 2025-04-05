package handler

import (
	"github.com/labstack/echo/v4"
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
	// User routes
	e.POST("/users", h.CreateUser)
	e.GET("/users/:id", h.GetUser)
	e.PUT("/users/:id", h.UpdateUser)
	e.DELETE("/users/:id", h.DeactivateUser)
	e.GET("/users", h.ListUsers)
	e.POST("/auth/login", h.Login)

	// Driver routes
	e.POST("/drivers", h.RegisterDriver)
	e.PUT("/drivers/:id/location", h.UpdateDriverLocation)
	e.PUT("/drivers/:id/availability", h.UpdateDriverAvailability)
	e.GET("/drivers/nearby", h.GetNearbyDrivers)
	e.PUT("/drivers/:id/verify", h.VerifyDriver)
}
