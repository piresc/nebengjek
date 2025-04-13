package handler

import (
	"github.com/labstack/echo/v4"
	rides "github.com/piresc/nebengjek/services/Rides"
)

// RideHandler handles Ride-related HTTP requests
type RideHandler struct {
	RideUC rides.RideUseCase
}

// NewRideHandler creates a new Ride handler
func NewRideHandler(RideUC rides.RideUseCase) *RideHandler {
	return &RideHandler{RideUC: RideUC}
}

// RegisterRoutes registers the Ride routes
func (h *RideHandler) RegisterRoutes(e *echo.Echo) {
	// Ride request routes
	e.POST("/Rides", h.CreateRideRequest)       // Create a new Ride request
	e.DELETE("/Rides/:id", h.CancelRideRequest) // Cancel a Ride request

	// Driver routes
	e.PUT("/Rides/:id/accept", h.AcceptRide)     // Driver accepts a Ride
	e.PUT("/Rides/:id/reject", h.RejectRide)     // Driver rejects a Ride
	e.PUT("/Rides/:id/start", h.StartRide)       // Driver starts a Ride
	e.PUT("/Rides/:id/complete", h.CompleteRide) // Driver completes a Ride

	// Ride status routes
	e.GET("/Rides/:id", h.GetRideStatus)                              // Get Ride status
	e.GET("/Rides/passenger/:id/active", h.GetActiveRideForPassenger) // Get active Ride for passenger
	e.GET("/Rides/driver/:id/active", h.GetActiveRideForDriver)       // Get active Ride for driver
	e.GET("/Rides/history", h.GetRideHistory)                         // Get Ride history

	// Rating routes
	e.POST("/Rides/:id/rate", h.RateRide) // Rate a Ride
}
