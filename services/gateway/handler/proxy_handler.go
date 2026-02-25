package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/services/gateway"
)

// ProxyHandler handles proxying requests to microservices following clean architecture
type ProxyHandler struct {
	gatewayUC gateway.GatewayUC
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(gatewayUC gateway.GatewayUC) *ProxyHandler {
	return &ProxyHandler{
		gatewayUC: gatewayUC,
	}
}

// ProxyToUsersService proxies requests to the Users Service
func (h *ProxyHandler) ProxyToUsersService(c echo.Context) error {
	return h.proxyRequest(c, "users-service")
}

// ProxyToMatchService proxies requests to the Match Service
func (h *ProxyHandler) ProxyToMatchService(c echo.Context) error {
	return h.proxyRequest(c, "match-service")
}

// ProxyToRidesService proxies requests to the Rides Service
func (h *ProxyHandler) ProxyToRidesService(c echo.Context) error {
	return h.proxyRequest(c, "rides-service")
}

// ProxyToLocationService proxies requests to the Location Service
func (h *ProxyHandler) ProxyToLocationService(c echo.Context) error {
	return h.proxyRequest(c, "location-service")
}

// proxyRequest handles the actual proxying logic using the UseCase layer
func (h *ProxyHandler) proxyRequest(c echo.Context, targetService string) error {
	// Get request details
	method := c.Request().Method
	path := c.Request().URL.Path

	// Clean path for internal routing
	path = h.cleanPath(path, targetService)

	// Read request body
	var body interface{}
	if c.Request().Body != nil {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
		}

		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON body")
			}
		}
	}

	// Extract query parameters
	queryParams := make(map[string]string)
	for key, values := range c.QueryParams() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Extract headers (excluding sensitive ones)
	headers := make(map[string]string)
	for key, values := range c.Request().Header {
		if len(values) > 0 && !h.isSensitiveHeader(key) {
			headers[key] = values[0]
		}
	}

	// Add user context if available (from JWT middleware)
	if userID := c.Get("user_id"); userID != nil {
		headers["X-User-ID"] = fmt.Sprintf("%v", userID)
	}
	if role := c.Get("role"); role != nil {
		headers["X-User-Role"] = fmt.Sprintf("%v", role)
	}

	// Call appropriate UseCase method based on target service
	var resp *core.ProxyResponse
	var err error

	switch targetService {
	case "users-service":
		resp, err = h.gatewayUC.ProxyToUsersService(c.Request().Context(), method, path, body, headers, queryParams)
	case "match-service":
		resp, err = h.gatewayUC.ProxyToMatchService(c.Request().Context(), method, path, body, headers, queryParams)
	case "rides-service":
		resp, err = h.gatewayUC.ProxyToRidesService(c.Request().Context(), method, path, body, headers, queryParams)
	case "location-service":
		resp, err = h.gatewayUC.ProxyToLocationService(c.Request().Context(), method, path, body, headers, queryParams)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Unknown target service")
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway,
			fmt.Sprintf("Failed to communicate with %s: %v", targetService, err))
	}

	for key, values := range resp.Headers {
		for _, value := range values {
			c.Response().Header().Add(key, value)
		}
	}

	// Return response
	return c.Blob(resp.StatusCode, "application/json", resp.Body)
}

// at file scope, add:
var servicePrefixes = map[string][]string{
	"users-service":    {"/api/v1/users", "/users"},
	"match-service":    {"/api/v1/match", "/match"},
	"rides-service":    {"/api/v1/rides", "/rides"},
	"location-service": {"/api/v1/location", "/location"},
}

func (h *ProxyHandler) cleanPath(path, service string) string {
	// Get prefixes for the specific service
	prefixes, ok := servicePrefixes[service]
	if !ok {
		// If service not found, try all prefixes
		for _, servicePrefixes := range servicePrefixes {
			prefixes = append(prefixes, servicePrefixes...)
		}
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// isSensitiveHeader checks if a header should be excluded from proxying
func (h *ProxyHandler) isSensitiveHeader(header string) bool {
	sensitiveHeaders := []string{
		"Authorization",
		"X-API-Key",
		"Cookie",
		"Set-Cookie",
		"Host",
		"Content-Length",
		"Transfer-Encoding",
		"Connection",
		"Upgrade",
	}

	headerLower := strings.ToLower(header)
	for _, sensitive := range sensitiveHeaders {
		if strings.ToLower(sensitive) == headerLower {
			return true
		}
	}
	return false
}
