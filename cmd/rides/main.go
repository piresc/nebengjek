package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/health"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/services/rides/gateway"
	"github.com/piresc/nebengjek/services/rides/handler"
	"github.com/piresc/nebengjek/services/rides/repository"
	"github.com/piresc/nebengjek/services/rides/usecase"
)

func main() {
	appName := "rides-service"
	configPath := "config/rides.env"
	configs := config.InitConfig(configPath)

	// Initialize New Relic and Zap logger
	nrApp := nrpkg.InitNewRelic(configs)

	zapLogger, err := logger.InitZapLoggerFromConfig(configs, nrApp)
	if err != nil {
		log.Fatalf("Failed to create Zap logger: %v", err)
	}
	defer zapLogger.Close()

	// Set global logger for application-wide access
	logger.SetGlobalLogger(zapLogger)

	// Log startup with global logger
	logger.Info("Starting application",
		logger.String("app", appName),
		logger.String("version", configs.App.Version),
		logger.String("environment", configs.App.Environment),
	)

	// Initialize PostgreSQL database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		zapLogger.Fatal("Failed to connect to PostgreSQL", logger.Err(err))
	}
	defer postgresClient.Close()

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(configs.Redis)
	if err != nil {
		zapLogger.Fatal("Failed to connect to Redis", logger.Err(err))
	}
	defer redisClient.Close()

	// Initialize JetStream-enabled NATS client
	natsClient, err := nats.NewClient(configs.NATS.URL)
	if err != nil {
		zapLogger.Fatal("Failed to connect to NATS with JetStream", logger.Err(err))
	}
	defer natsClient.Close()

	// Verify JetStream is available
	if !natsClient.IsConnected() {
		zapLogger.Fatal("NATS JetStream client not connected")
	}

	logger.Info("JetStream client initialized successfully",
		logger.String("url", configs.NATS.URL),
		logger.Bool("connected", natsClient.IsConnected()))

	// Initialize repository
	rideRepo := repository.NewRideRepository(configs, postgresClient.GetDB())

	// Initialize gateway
	ridesGW := gateway.NewRideGW(natsClient)

	// Initialize usecase
	rideUC, err := usecase.NewRideUC(configs, rideRepo, ridesGW)
	if err != nil {
		zapLogger.Fatal("Failed to initialize ride use case", logger.Err(err))
	}

	// Initialize handlers
	rideHandler := handler.NewHandler(rideUC, natsClient, configs)

	// Initialize NATS consumers
	if err := rideHandler.InitNATSConsumers(); err != nil {
		zapLogger.Fatal("Failed to initialize NATS consumers", logger.Err(err))
	}

	// Initialize Echo server
	e := echo.New()

	// Add middlewares (panic recovery should be first)
	e.Use(middleware.PanicRecoveryWithZapMiddleware(zapLogger))
	e.Use(middleware.RequestIDMiddleware())
	e.Use(logger.ZapEchoMiddleware(zapLogger))

	// Initialize API key middleware
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(&configs.APIKey)

	// Initialize enhanced health service
	healthService := health.NewHealthService(zapLogger)
	healthService.AddChecker("postgres", health.NewPostgresHealthChecker(postgresClient))
	healthService.AddChecker("redis", health.NewRedisHealthChecker(redisClient))
	healthService.AddChecker("nats", health.NewNATSHealthChecker(natsClient))

	// Register enhanced health endpoints
	health.RegisterEnhancedHealthEndpoints(e, appName, configs.App.Version, healthService)

	// Register service routes
	rideHandler.RegisterRoutes(e, apiKeyMiddleware)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", configs.Server.Port)
		zapLogger.Info("Starting HTTP server",
			logger.String("address", addr),
			logger.String("app", appName))

		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Failed to start server", logger.Err(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt signal
	sig := <-quit
	zapLogger.Info("Received shutdown signal", logger.String("signal", sig.String()))

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	zapLogger.Info("Shutting down HTTP server...")
	if err := e.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", logger.Err(err))
	}

	// Close PostgreSQL connection
	zapLogger.Info("Closing PostgreSQL connection...")
	postgresClient.Close()

	// Close Redis connection
	zapLogger.Info("Closing Redis connection...")
	if err := redisClient.Close(); err != nil {
		zapLogger.Error("Error closing Redis connection", logger.Err(err))
	}

	// Close NATS connection
	zapLogger.Info("Closing NATS connection...")
	natsClient.Close()

	// Shutdown New Relic
	if nrApp != nil {
		zapLogger.Info("Shutting down New Relic...")
		nrApp.Shutdown(10 * time.Second)
	}

	// Sync and close logger
	zapLogger.Info("Server exiting gracefully")
	_ = zapLogger.Sync()
}
