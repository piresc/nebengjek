package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location"
)

// LocationHandler handles location-related HTTP requests
type LocationHandler struct {
	locationUC location.LocationUseCase
}

// NewLocationHandler creates a new location handler
func NewLocationHandler(locationUC location.LocationUseCase) *LocationHandler {
	return &LocationHandler{locationUC: locationUC}
}

// RegisterRoutes registers the location routes
func (h *LocationHandler) RegisterRoutes(e *echo.Echo) {
	// Location routes as per README specifications
	e.POST("/locations", h.UpdateDriverLocation)   // Stores driver/customer coordinates
	e.GET("/locations/nearby", h.GetNearbyDrivers) // Finds nearby drivers using PostGIS

	// Additional routes
	e.PUT("/drivers/:id/availability", h.UpdateDriverAvailability)

	// Periodic location update routes
	e.POST("/locations/periodic/start", h.StartPeriodicUpdates) // Start periodic location updates
	e.POST("/locations/periodic/stop", h.StopPeriodicUpdates)   // Stop periodic location updates

	// Event-based location update routes
	e.POST("/locations/event", h.UpdateLocationOnEvent) // Update location based on an event

	// Customer location routes
	e.POST("/locations/customer", h.UpdateCustomerLocation) // Update customer location

	// Location history routes
	e.GET("/locations/history", h.GetLocationHistory) // Get location history for a user
}

// UpdateDriverLocation handles driver location update requests
func (h *LocationHandler) UpdateDriverLocation(c echo.Context) error {
	// Get driver ID from path parameter or query parameter
	id := c.Param("id")
	if id == "" {
		id = c.QueryParam("driver_id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "driver ID is required"})
		}
	}

	// Parse location from request body
	var location models.Location
	if err := c.Bind(&location); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid location data"})
	}

	// Update driver location
	if err := h.locationUC.UpdateDriverLocation(c.Request().Context(), id, &location); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "driver location updated successfully"})
}

// UpdateDriverAvailability handles driver availability update requests
func (h *LocationHandler) UpdateDriverAvailability(c echo.Context) error {
	// Get driver ID from path parameter
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "driver ID is required"})
	}

	// Parse availability from request body
	type AvailabilityRequest struct {
		IsAvailable bool `json:"is_available"`
	}
	var availability AvailabilityRequest
	if err := c.Bind(&availability); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid availability data"})
	}

	// Update driver availability
	if err := h.locationUC.UpdateDriverAvailability(c.Request().Context(), id, availability.IsAvailable); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "driver availability updated successfully"})
}

// GetNearbyDrivers handles nearby drivers retrieval requests
func (h *LocationHandler) GetNearbyDrivers(c echo.Context) error {
	// Parse location parameters from query string
	latStr := c.QueryParam("latitude")
	lngStr := c.QueryParam("longitude")
	radiusStr := c.QueryParam("radius")

	if latStr == "" || lngStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "latitude and longitude are required"})
	}

	// Parse latitude
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid latitude"})
	}

	// Parse longitude
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid longitude"})
	}

	// Parse radius (default to 1 km if not provided)
	radius := 1.0
	if radiusStr != "" {
		radius, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid radius"})
		}
	}

	// Create location object
	location := &models.Location{
		Latitude:  lat,
		Longitude: lng,
		Timestamp: c.Request().Context().Value("requestTime").(time.Time),
	}

	// Get nearby drivers
	drivers, err := h.locationUC.GetNearbyDrivers(c.Request().Context(), location, radius)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, drivers)
}

// UpdateCustomerLocation handles customer location update requests
func (h *LocationHandler) UpdateCustomerLocation(c echo.Context) error {
	// Get customer ID from query parameter
	id := c.QueryParam("customer_id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "customer ID is required"})
	}

	// Parse location from request body
	var location models.Location
	if err := c.Bind(&location); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid location data"})
	}

	// Update customer location
	if err := h.locationUC.UpdateCustomerLocation(c.Request().Context(), id, &location); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "customer location updated successfully"})
}

// StartPeriodicUpdates handles requests to start periodic location updates
func (h *LocationHandler) StartPeriodicUpdates(c echo.Context) error {
	// Parse request body
	type PeriodicUpdateRequest struct {
		UserID   string `json:"user_id"`
		Role     string `json:"role"` // driver or customer
		Interval int    `json:"interval_seconds"`
	}

	var req PeriodicUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request data"})
	}

	// Validate request
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user ID is required"})
	}

	if req.Role != "driver" && req.Role != "customer" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role must be 'driver' or 'customer'"})
	}

	// Set default interval if not provided or out of range
	interval := time.Duration(req.Interval) * time.Second
	if interval < 30*time.Second || interval > 60*time.Second {
		// Default to 30 seconds if not in range
		interval = 30 * time.Second
	}

	// Start periodic updates
	if err := h.locationUC.StartPeriodicUpdates(c.Request().Context(), req.UserID, req.Role, interval); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message":  "periodic location updates started",
		"interval": interval.String(),
	})
}

// StopPeriodicUpdates handles requests to stop periodic location updates
func (h *LocationHandler) StopPeriodicUpdates(c echo.Context) error {
	// Parse request body
	type StopUpdateRequest struct {
		UserID string `json:"user_id"`
	}

	var req StopUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request data"})
	}

	// Validate request
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user ID is required"})
	}

	// Stop periodic updates
	if err := h.locationUC.StopPeriodicUpdates(c.Request().Context(), req.UserID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "periodic location updates stopped"})
}

// UpdateLocationOnEvent handles event-based location update requests
func (h *LocationHandler) UpdateLocationOnEvent(c echo.Context) error {
	// Parse request body
	type EventUpdateRequest struct {
		UserID    string          `json:"user_id"`
		Role      string          `json:"role"` // driver or customer
		Location  models.Location `json:"location"`
		EventType string          `json:"event_type"` // app_state_change, ride_request, etc.
	}

	var req EventUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request data"})
	}

	// Validate request
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user ID is required"})
	}

	if req.Role != "driver" && req.Role != "customer" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role must be 'driver' or 'customer'"})
	}

	if req.EventType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event type is required"})
	}

	// Update location based on event
	if err := h.locationUC.UpdateLocationOnEvent(c.Request().Context(), req.UserID, req.Role, &req.Location, req.EventType); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "location updated based on event"})
}

// GetLocationHistory handles requests to retrieve location history
func (h *LocationHandler) GetLocationHistory(c echo.Context) error {
	// Get parameters from query string
	userID := c.QueryParam("user_id")
	startTimeStr := c.QueryParam("start_time")
	endTimeStr := c.QueryParam("end_time")

	// Validate parameters
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user ID is required"})
	}

	// Parse start time (default to 24 hours ago if not provided)
	var startTime time.Time
	if startTimeStr == "" {
		startTime = time.Now().Add(-24 * time.Hour)
	} else {
		var err error
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start time format, use RFC3339"})
		}
	}

	// Parse end time (default to now if not provided)
	var endTime time.Time
	if endTimeStr == "" {
		endTime = time.Now()
	} else {
		var err error
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end time format, use RFC3339"})
		}
	}

	// Get location history
	locations, err := h.locationUC.GetLocationHistory(c.Request().Context(), userID, startTime, endTime)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, locations)
}
