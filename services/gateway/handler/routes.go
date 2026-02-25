package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
)

// RegisterRoutes registers all protocol handlers and their routes
func (h *Handler) RegisterRoutes(e *echo.Echo, Middleware *middleware.Middleware) {
	// WebSocket routes - use custom WebSocket JWT middleware
	wsGroup := e.Group("/ws", h.GetWebSocketJWTMiddleware())
	wsGroup.GET("", h.wsHandler.HandleWebSocket)

	// WebSocket health monitoring endpoints - API key protected
	health := e.Group("/health/websocket", Middleware.APIKeyHandler("gateway-service"))
	health.GET("/stats", h.GetWebSocketHealthStats)
	health.GET("/connections", h.GetAllWebSocketConnections)
	health.GET("/connections/:user_id", h.GetWebSocketConnectionHealth)
	health.GET("/connections/healthy/count", h.GetHealthyConnectionsCount)

	// Public API routes (proxy to microservices)
	h.registerPublicRoutes(e, Middleware)

}

// registerPublicRoutes registers public API routes that proxy to microservices
func (h *Handler) registerPublicRoutes(e *echo.Echo, mw *middleware.Middleware) {
	// Authentication routes (proxy to Users Service) - No JWT required
	auth := e.Group("/auth")
	auth.POST("/otp/generate", h.proxyHandler.ProxyToUsersService)
	auth.POST("/otp/verify", h.proxyHandler.ProxyToUsersService)

	// User routes (proxy to Users Service) - JWT required
	users := e.Group("/users", mw.JWTHandler())
	users.POST("", h.proxyHandler.ProxyToUsersService)
	users.GET("/:id", h.proxyHandler.ProxyToUsersService)
	users.GET("/profile", h.proxyHandler.ProxyToUsersService)
	users.PUT("/profile", h.proxyHandler.ProxyToUsersService)

	// Driver routes (proxy to Users Service) - JWT required
	drivers := e.Group("/drivers", mw.JWTHandler())
	drivers.POST("/register", h.proxyHandler.ProxyToUsersService)
	drivers.PUT("/status", h.proxyHandler.ProxyToUsersService)

	// Match routes (proxy to Match Service) - JWT required
	matches := e.Group("/matches", mw.JWTHandler())
	matches.POST("/request", h.proxyHandler.ProxyToMatchService)
	matches.GET("/:id", h.proxyHandler.ProxyToMatchService)
	matches.PUT("/:id/accept", h.proxyHandler.ProxyToMatchService)
	matches.PUT("/:id/reject", h.proxyHandler.ProxyToMatchService)

	// Ride routes (proxy to Rides Service) - JWT required
	rides := e.Group("/rides", mw.JWTHandler())
	rides.POST("", h.proxyHandler.ProxyToRidesService)
	rides.GET("/:id", h.proxyHandler.ProxyToRidesService)
	rides.PUT("/:id/start", h.proxyHandler.ProxyToRidesService)
	rides.PUT("/:id/complete", h.proxyHandler.ProxyToRidesService)
	rides.PUT("/:id/cancel", h.proxyHandler.ProxyToRidesService)

	// Location routes (proxy to Location Service) - JWT required
	locations := e.Group("/locations", mw.JWTHandler())
	locations.POST("/update", h.proxyHandler.ProxyToLocationService)
	locations.GET("/nearby", h.proxyHandler.ProxyToLocationService)

	// Payment routes (proxy to Rides Service) - JWT required
	payments := e.Group("/payments", mw.JWTHandler())
	payments.POST("/process", h.proxyHandler.ProxyToRidesService)
	payments.GET("/:id", h.proxyHandler.ProxyToRidesService)
}

// WebSocket Health Monitoring Endpoints

// GetWebSocketHealthStats returns overall WebSocket connection statistics
func (h *Handler) GetWebSocketHealthStats(c echo.Context) error {
	stats := h.wsHandler.GetHeartbeatStats()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// GetAllWebSocketConnections returns health status for all WebSocket connections
func (h *Handler) GetAllWebSocketConnections(c echo.Context) error {
	connections := h.wsHandler.GetAllConnectionsHealth()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    connections,
	})
}

// GetWebSocketConnectionHealth returns health status for a specific user's WebSocket connection
func (h *Handler) GetWebSocketConnectionHealth(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id parameter is required")
	}

	health := h.wsHandler.GetConnectionHealth(userID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    health,
	})
}

// GetHealthyConnectionsCount returns the count of healthy WebSocket connections
func (h *Handler) GetHealthyConnectionsCount(c echo.Context) error {
	count := h.wsHandler.GetHealthyConnectionsCount()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"healthy_connections_count": count,
		},
	})
}
