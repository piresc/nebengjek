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
	newrelictracer "github.com/piresc/nebengjek/internal/pkg/tracing/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
	"github.com/piresc/nebengjek/services/notification/handler"
	httpHandler "github.com/piresc/nebengjek/services/notification/handler/http"
	natsHandler "github.com/piresc/nebengjek/services/notification/handler/nats"
	"github.com/piresc/nebengjek/services/notification/repository"
	"github.com/piresc/nebengjek/services/notification/usecase"
)

func main() {
	appName := "notification-service"
	configPath := "/Users/pirescerullo/GitHub/assessment/nebengjek/config/notification.env"
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

	// Initialize unified tracer
	tracer := newrelictracer.NewTracer(&tracing.Config{
		Enabled:     configs.NewRelic.Enabled,
		ServiceName: appName,
	}, nrApp)

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
	// Wrap PostgreSQL client with tracing (not used in this service)

	// Initialize JetStream-enabled NATS client
	natsClient, err := nats.NewClient(configs.NATS.URL)
	if err != nil {
		slogLogger.Error("Failed to connect to NATS with JetStream", slog.Any("error", err))
		os.Exit(1)
	}
	// Wrap NATS client with tracing
	tracedNatsClient := nats.NewTracedClient(natsClient, tracer)

	// Verify JetStream is available
	if !natsClient.IsConnected() {
		slogLogger.Error("NATS JetStream client not connected")
		os.Exit(1)
	}

	slogLogger.Info("JetStream client initialized successfully",
		slog.String("url", configs.NATS.URL),
		slog.Bool("connected", natsClient.IsConnected()))

	// Configure WebSocket streams for multi-gateway broadcasting
	slogLogger.Info("Configuring WebSocket streams for multi-gateway broadcasting...")
	if err := nats.ConfigureWebSocketStreams(context.Background(), tracedNatsClient.GetClient()); err != nil {
		slogLogger.Error("Failed to configure WebSocket streams", slog.Any("error", err))
		os.Exit(1)
	}
	slogLogger.Info("WebSocket streams configured successfully")

	// Initialize repository with traced client
	notificationRepo := repository.NewNotificationRepo(postgresClient.GetDB())

	// Initialize usecase with traced client
	notificationUC := usecase.NewNotificationUC(notificationRepo, tracedNatsClient.GetClient(), slogLogger)

	// Initialize HTTP handler
	notificationHTTPHandler := httpHandler.NewNotificationHandler(notificationUC)

	// Initialize NATS handler for consuming events
	notificationNATSHandler := natsHandler.NewNotificationHandler(notificationUC, tracedNatsClient.GetClient(), slogLogger)

	// Start NATS consumers
	slogLogger.Info("Initializing NATS consumers for notification service...")
	if err := notificationNATSHandler.InitNATSConsumers(); err != nil {
		slogLogger.Error("Failed to initialize NATS consumers", slog.Any("error", err))
		os.Exit(1)
	}
	slogLogger.Info("NATS consumers initialized successfully")

	// Initialize Echo server
	e := echo.New()

	// Initialize enhanced health service
	healthService := health.NewHealthService(slogLogger)
	healthService.AddChecker("postgres", health.NewPostgresHealthChecker(postgresClient))
	healthService.AddChecker("nats", health.NewNATSHealthChecker(natsClient))

	// Initialize middleware with tracing
	MW := middleware.NewMiddleware(configs, slogLogger, tracer)
	tracingMiddleware := middleware.NewTracingMiddleware(&tracing.Config{
		Enabled:     configs.NewRelic.Enabled,
		ServiceName: appName,
	}, tracer)

	// Register enhanced health endpoints BEFORE applying middleware
	health.RegisterEnhancedHealthEndpoints(e, appName, configs.App.Version, healthService)

	// Register additional health endpoint for /health/notification
	healthGroup := e.Group("/health")
	healthGroup.GET("/notification", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	e.Use(tracingMiddleware.Handler())
	e.Use(MW.Handler())

	// Register service routes
	handler.RegisterRoutes(e, notificationHTTPHandler)

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