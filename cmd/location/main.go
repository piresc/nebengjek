package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/health"
	slogpkg "github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/observability"
	"github.com/piresc/nebengjek/services/location/gateway"
	"github.com/piresc/nebengjek/services/location/handler"
	"github.com/piresc/nebengjek/services/location/repository"
	"github.com/piresc/nebengjek/services/location/usecase"
)

func main() {
	appName := "location-service"
	configPath := "config/location.env"
	configs := config.InitConfig(configPath)

	// Initialize New Relic
	nrApp := nrpkg.InitNewRelic(configs)

	// Initialize slog logger with New Relic integration
	slogLogger := slogpkg.NewSlogLogger(slogpkg.SlogConfig{
		Level:       slog.LevelInfo,
		ServiceName: appName,
		NewRelic:    nrApp,
		Format:      "json",
	})

	// Initialize observability tracer
	tracerFactory := observability.NewTracerFactory()
	tracer := tracerFactory.CreateTracer(nrApp)

	// Log startup
	slogLogger.Info("Starting application",
		slog.String("app", appName),
		slog.String("version", configs.App.Version),
		slog.String("environment", configs.App.Environment),
	)

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(configs.Redis)
	if err != nil {
		slogLogger.Error("Failed to connect to Redis", slog.Any("error", err))
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize JetStream-enabled NATS client
	natsClient, err := nats.NewClient(configs.NATS.URL)
	if err != nil {
		slogLogger.Error("Failed to connect to NATS with JetStream", slog.Any("error", err))
		os.Exit(1)
	}
	defer natsClient.Close()

	// Verify JetStream is available
	if !natsClient.IsConnected() {
		slogLogger.Error("NATS JetStream client not connected")
		os.Exit(1)
	}

	slogLogger.Info("JetStream client initialized successfully",
		slog.String("url", configs.NATS.URL),
		slog.Bool("connected", natsClient.IsConnected()))

	// Initialize repository
	locationRepo := repository.NewLocationRepository(redisClient, configs)

	// Initialize gateway
	locationGW := gateway.NewLocationGW(natsClient)

	// Initialize usecase
	locationUC := usecase.NewLocationUC(locationRepo, locationGW)

	// Initialize handlers
	locationHandler := handler.NewHTTPHandler(locationUC, natsClient, configs, nrApp)

	// Initialize NATS consumers
	if err := locationHandler.InitNATSConsumers(); err != nil {
		slogLogger.Error("Failed to initialize NATS consumers", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize Echo server
	e := echo.New()

	// Initialize enhanced health service
	healthService := health.NewHealthService(slogLogger)
	healthService.AddChecker("redis", health.NewRedisHealthChecker(redisClient))
	healthService.AddChecker("nats", health.NewNATSHealthChecker(natsClient))

	// Initialize middleware
	MW := middleware.NewMiddleware(middleware.Config{
		Logger: slogLogger,
		Tracer: tracer,
		APIKeys: map[string]string{
			"user-service":     configs.APIKey.UserService,
			"match-service":    configs.APIKey.MatchService,
			"rides-service":    configs.APIKey.RidesService,
			"location-service": configs.APIKey.LocationService,
		},
		ServiceName: appName,
	})

	// Register enhanced health endpoints BEFORE applying middleware
	health.RegisterEnhancedHealthEndpoints(e, appName, configs.App.Version, healthService)

	// Register additional health endpoint for /health/location
	healthGroup := e.Group("/health")
	healthGroup.GET("/location", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	e.Use(MW.Handler())

	// Register service routes
	locationHandler.RegisterRoutes(e, MW)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", configs.Server.Port)
		slogLogger.Info("Starting HTTP server",
			slog.String("address", addr),
			slog.String("app", appName))

		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slogLogger.Error("Failed to start server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt signal
	sig := <-quit
	slogLogger.Info("Received shutdown signal", slog.String("signal", sig.String()))

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	slogLogger.Info("Shutting down HTTP server...")
	if err := e.Shutdown(ctx); err != nil {
		slogLogger.Error("Server forced to shutdown", slog.Any("error", err))
	}

	// Close Redis connection
	slogLogger.Info("Closing Redis connection...")
	if err := redisClient.Close(); err != nil {
		slogLogger.Error("Error closing Redis connection", slog.Any("error", err))
	}

	// Close NATS connection
	slogLogger.Info("Closing NATS connection...")
	natsClient.Close()

	// Shutdown New Relic
	if nrApp != nil {
		slogLogger.Info("Shutting down New Relic...")
		nrApp.Shutdown(10 * time.Second)
	}

	// Final log
	slogLogger.Info("Server exiting gracefully")
}
