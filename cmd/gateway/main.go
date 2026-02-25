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
	middlewaretracing "github.com/piresc/nebengjek/internal/pkg/middleware/tracing"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
	newrelictracer "github.com/piresc/nebengjek/internal/pkg/tracing/newrelic"
	gatewaygateway "github.com/piresc/nebengjek/services/gateway/gateway"
	"github.com/piresc/nebengjek/services/gateway/handler"
	gatewaynats "github.com/piresc/nebengjek/services/gateway/handler/nats"
	gatewaywebsocket "github.com/piresc/nebengjek/services/gateway/handler/websocket"
	gatewayrepo "github.com/piresc/nebengjek/services/gateway/repository"
	gatewayusecase "github.com/piresc/nebengjek/services/gateway/usecase"
	userrepo "github.com/piresc/nebengjek/services/users/repository"
	useruc "github.com/piresc/nebengjek/services/users/usecase"
)

var (
	Version   = "development"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	appName := "gateway-service"
	configPath := "config/gateway.env"
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
		slog.String("version", Version),
		slog.String("environment", configs.App.Environment),
	)

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(configs.Redis)
	if err != nil {
		slogLogger.Error("Failed to connect to Redis", slog.Any("error", err))
		os.Exit(1)
	}
	defer redisClient.Close()

	// Wrap Redis client with tracing
	tracedRedisClient := database.NewTracedRedisClient(redisClient, tracer)

	// Initialize JetStream-enabled NATS client for notification consumption
	natsClient, err := nats.NewClient(configs.NATS.URL)
	if err != nil {
		slogLogger.Error("Failed to connect to NATS with JetStream", slog.Any("error", err))
		os.Exit(1)
	}
	defer natsClient.Close()

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

	// Generate unique server ID for multi-gateway support
	hostname, _ := os.Hostname()
	serverID := fmt.Sprintf("gateway-%s-%d", hostname, time.Now().Unix())

	// Initialize repositories
	userRepo := userrepo.NewUserRepo(configs, nil, tracedRedisClient.RedisClient)
	sessionRepo := gatewayrepo.NewWSSessionRepository(tracedRedisClient.RedisClient, serverID)

	// Initialize gateways
	gatewayGW := gatewaygateway.NewHTTPGateway(configs)

	// Initialize usecases
	userUC := useruc.NewUserUC(userRepo, nil, configs)
	gatewayUC := gatewayusecase.NewGatewayUC(userUC, gatewayGW)

	// Initialize handlers with multi-gateway support
	wsHandler := gatewaywebsocket.NewEchoWebSocketHandler(gatewayUC, userUC, sessionRepo, serverID)
	gatewayHandler := handler.NewHandler(gatewayUC, configs, nrApp, wsHandler, sessionRepo, serverID)

	// Initialize notification handler for multi-gateway broadcasting
	notificationHandler := gatewaynats.NewGatewayNotificationHandler(tracedNatsClient.GetClient(), wsHandler, sessionRepo, slogLogger, serverID)

	// Initialize NATS consumers for notifications
	slogLogger.Info("Initializing NATS consumers for gateway...")
	if err := notificationHandler.InitNotificationConsumer(); err != nil {
		slogLogger.Error("Failed to initialize notification NATS consumer", slog.Any("error", err))
		os.Exit(1)
	}
	slogLogger.Info("NATS notification consumer initialized successfully")

	// Configure WebSocket streams for multi-gateway broadcasting
	slogLogger.Info("Configuring WebSocket streams for multi-gateway broadcasting...")
	if err := natspkg.ConfigureWebSocketStreams(context.Background(), tracedNatsClient.GetClient()); err != nil {
		slogLogger.Error("Failed to configure WebSocket streams", slog.Any("error", err))
		os.Exit(1)
	}
	slogLogger.Info("WebSocket streams configured successfully")

	// Initialize multi-gateway consumers for WebSocket broadcasting
	slogLogger.Info("Initializing multi-gateway WebSocket consumers...")
	if err := notificationHandler.InitMultiGatewayConsumers(); err != nil {
		slogLogger.Error("Failed to initialize multi-gateway consumers", slog.Any("error", err))
		os.Exit(1)
	}
	slogLogger.Info("Multi-gateway WebSocket consumers initialized successfully")


	// Initialize Echo server
	e := echo.New()

	// Initialize enhanced health service
	healthService := health.NewHealthService(slogLogger)
	healthService.AddChecker("redis", health.NewRedisHealthChecker(redisClient))
	healthService.AddChecker("nats", health.NewNATSHealthChecker(natsClient))

	// Initialize middleware with tracing
	MW := middleware.NewMiddleware(configs, slogLogger, tracer)
	tracingMiddleware := middlewaretracing.NewTracingMiddleware(&tracing.Config{
		Enabled:     configs.NewRelic.Enabled,
		ServiceName: appName,
	}, tracer)

	// Register enhanced health endpoints BEFORE applying middleware
	health.RegisterEnhancedHealthEndpoints(e, appName, Version, healthService)

	// Register additional health endpoint for /health/gateway
	healthGroup := e.Group("/health")
	healthGroup.GET("/gateway", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	e.Use(tracingMiddleware.Handler())
	e.Use(MW.Handler())

	// Register service routes
	gatewayHandler.RegisterRoutes(e, MW)

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
