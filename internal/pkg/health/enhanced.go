package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/nats"
)

// HealthChecker defines the interface for health checking dependencies
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}

// PostgresHealthChecker checks PostgreSQL connection health
type PostgresHealthChecker struct {
	client *database.PostgresClient
}

// NewPostgresHealthChecker creates a new PostgreSQL health checker
func NewPostgresHealthChecker(client *database.PostgresClient) *PostgresHealthChecker {
	return &PostgresHealthChecker{client: client}
}

// CheckHealth checks if PostgreSQL is healthy
func (p *PostgresHealthChecker) CheckHealth(ctx context.Context) error {
	if p.client == nil {
		return nil // Skip if no PostgreSQL client
	}
	return p.client.GetDB().PingContext(ctx)
}

// RedisHealthChecker checks Redis connection health
type RedisHealthChecker struct {
	client *database.RedisClient
}

// NewRedisHealthChecker creates a new Redis health checker
func NewRedisHealthChecker(client *database.RedisClient) *RedisHealthChecker {
	return &RedisHealthChecker{client: client}
}

// CheckHealth checks if Redis is healthy
func (r *RedisHealthChecker) CheckHealth(ctx context.Context) error {
	if r.client == nil {
		return nil // Skip if no Redis client
	}
	return r.client.Client.Ping(ctx).Err()
}

// NATSHealthChecker checks NATS connection health
type NATSHealthChecker struct {
	client *nats.Client
}

// NewNATSHealthChecker creates a new NATS health checker
func NewNATSHealthChecker(client *nats.Client) *NATSHealthChecker {
	return &NATSHealthChecker{client: client}
}

// CheckHealth checks if NATS and JetStream are healthy
func (n *NATSHealthChecker) CheckHealth(ctx context.Context) error {
	if n.client == nil {
		return nil // Skip if no NATS client
	}

	// Check basic NATS connection
	conn := n.client.GetConn()
	if conn == nil || !conn.IsConnected() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "NATS not connected")
	}

	// Check JetStream availability
	js := n.client.GetJetStream()
	if js == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "JetStream not available")
	}

	// Verify we can list streams (basic JetStream health check)
	_, err := n.client.ListStreams()
	if err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "JetStream streams not accessible: "+err.Error())
	}

	return nil
}

// HealthService manages health checks for multiple dependencies
type HealthService struct {
	checkers map[string]HealthChecker
	logger   *slog.Logger
}

// NewHealthService creates a new health service
func NewHealthService(slogLogger *slog.Logger) *HealthService {
	return &HealthService{
		checkers: make(map[string]HealthChecker),
		logger:   slogLogger,
	}
}

// AddChecker registers a health checker for a dependency
func (h *HealthService) AddChecker(name string, checker HealthChecker) {
	h.checkers[name] = checker
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string                    `json:"status"`
	Timestamp    time.Time                 `json:"timestamp"`
	Service      string                    `json:"service"`
	Version      string                    `json:"version,omitempty"`
	Dependencies map[string]DependencyInfo `json:"dependencies"`
}

// DependencyInfo represents health info for a dependency
type DependencyInfo struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// CheckAllHealth performs health checks on all registered dependencies
func (h *HealthService) CheckAllHealth(ctx context.Context) HealthResponse {
	response := HealthResponse{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Dependencies: make(map[string]DependencyInfo),
	}

	for name, checker := range h.checkers {
		if err := checker.CheckHealth(ctx); err != nil {
			h.logger.Error("Health check failed",
				logger.String("dependency", name),
				logger.Err(err))

			response.Dependencies[name] = DependencyInfo{
				Status: "unhealthy",
				Error:  err.Error(),
			}
			response.Status = "unhealthy"
		} else {
			response.Dependencies[name] = DependencyInfo{
				Status: "healthy",
			}
		}
	}

	return response
}

// RegisterEnhancedHealthEndpoints registers comprehensive health check endpoints
func RegisterEnhancedHealthEndpoints(e *echo.Echo, serviceName, version string, healthService *HealthService) {
	healthGroup := e.Group("/health")

	// Basic health check (for load balancers)
	healthGroup.GET("", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"service":   serviceName,
			"timestamp": time.Now(),
		})
	})

	// Detailed health check with dependencies
	healthGroup.GET("/detailed", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
		defer cancel()

		response := healthService.CheckAllHealth(ctx)
		response.Service = serviceName
		response.Version = version

		statusCode := http.StatusOK
		if response.Status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		return c.JSON(statusCode, response)
	})

	// Readiness probe (for Kubernetes)
	healthGroup.GET("/ready", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
		defer cancel()

		response := healthService.CheckAllHealth(ctx)
		response.Service = serviceName

		if response.Status == "unhealthy" {
			return c.JSON(http.StatusServiceUnavailable, response)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "ready",
			"service": serviceName,
		})
	})

	// Liveness probe (for Kubernetes)
	healthGroup.GET("/live", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "alive",
			"service": serviceName,
		})
	})
}
