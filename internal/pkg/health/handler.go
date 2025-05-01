package health

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
)

// BuildInfo contains information about the build
type BuildInfo struct {
	Version     string    `json:"version"`
	GitCommit   string    `json:"git_commit"`
	BuildTime   string    `json:"build_time"`
	ServiceName string    `json:"service_name"`
	GoVersion   string    `json:"go_version"`
	Hostname    string    `json:"hostname"`
	ServerTime  time.Time `json:"server_time"`
}

// DefaultBuildInfo contains default build information
var DefaultBuildInfo = BuildInfo{
	Version:   "development",
	GitCommit: "unknown",
	BuildTime: "unknown",
	GoVersion: runtime.Version(),
}

// NewPingHandler creates a handler for the ping endpoint
func NewPingHandler(serviceName string) echo.HandlerFunc {
	// Try to get hostname for the response
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	buildInfo := DefaultBuildInfo
	buildInfo.ServiceName = serviceName

	// Use environment variables if available
	if version := os.Getenv("VERSION"); version != "" {
		buildInfo.Version = version
	}
	if gitCommit := os.Getenv("GIT_COMMIT"); gitCommit != "" {
		buildInfo.GitCommit = gitCommit
	}
	if buildTime := os.Getenv("BUILD_TIME"); buildTime != "" {
		buildInfo.BuildTime = buildTime
	}

	return func(c echo.Context) error {
		// Update dynamic information
		buildInfo.Hostname = hostname
		buildInfo.ServerTime = time.Now()
		
		return c.JSON(http.StatusOK, buildInfo)
	}
}

// RegisterHealthEndpoints registers the health check endpoints
func RegisterHealthEndpoints(e *echo.Echo, serviceName string) {
	// Basic ping endpoint
	e.GET("/ping", NewPingHandler(serviceName))
	
	// Kubernetes standard health endpoints
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
	
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
	
	e.GET("/ready", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
}