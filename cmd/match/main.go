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
	"github.com/piresc/nebengjek/services/match/gateway"
	"github.com/piresc/nebengjek/services/match/handler"
	"github.com/piresc/nebengjek/services/match/repository"
	"github.com/piresc/nebengjek/services/match/usecase"
)

func main() {
	appName := "match-service"
	configPath := "config/match.env"
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

	// Initialize PostgreSQL database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		slogLogger.Error("Failed to connect to PostgreSQL", slog.Any("error", err))
		os.Exit(1)
	}
	defer postgresClient.Close()

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

	// Initialize repositories
	matchRepo := repository.NewMatchRepository(configs, postgresClient.GetDB(), redisClient)

	// Initialize  gateway with tracer and logger
	matchGW := gateway.NewMatchGW(natsClient, configs.Services.LocationServiceURL, &configs.APIKey, tracer, slogLogger)

	// Initialize usecase
	matchUC := usecase.NewMatchUC(configs, matchRepo, matchGW)

	// Initialize handlers
	handler := handler.NewHandler(matchUC, natsClient, nrApp)

	// Initialize NATS consumers
	if err := handler.InitNATSConsumers(); err != nil {
		slogLogger.Error("Failed to initialize NATS consumers", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize Echo server
	e := echo.New()

	// Use  middleware
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

	e.Use(MW.Handler())

	// Initialize enhanced health service
	healthService := health.NewHealthService(nil) // Pass nil for old logger since we're using slog
	healthService.AddChecker("postgres", health.NewPostgresHealthChecker(postgresClient))
	healthService.AddChecker("redis", health.NewRedisHealthChecker(redisClient))
	healthService.AddChecker("nats", health.NewNATSHealthChecker(natsClient))

	// Register enhanced health endpoints
	health.RegisterEnhancedHealthEndpoints(e, appName, configs.App.Version, healthService)

	// Create the old API key middleware for compatibility
	// Use  middleware instead of separate API key middleware

	// Register service routes
	handler.RegisterRoutes(e, MW)

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

	// Close PostgreSQL connection
	slogLogger.Info("Closing PostgreSQL connection...")
	postgresClient.Close()

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
