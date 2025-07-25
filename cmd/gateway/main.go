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
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/observability"
	gatewaygateway "github.com/piresc/nebengjek/services/gateway/gateway"
	"github.com/piresc/nebengjek/services/gateway/handler"
	gatewaywebsocket "github.com/piresc/nebengjek/services/gateway/handler/websocket"
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
	configPath := "/Users/pirescerullo/GitHub/assessment/nebengjek/config/gateway.env"
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

	// Initialize repositories
	userRepo := userrepo.NewUserRepo(configs, nil, redisClient)

	// Initialize gateways
	gatewayGW := gatewaygateway.NewHTTPGateway(configs)

	// Initialize usecases
	userUC := useruc.NewUserUC(userRepo, nil, configs)
	gatewayUC := gatewayusecase.NewGatewayUC(userUC, gatewayGW)

	// Initialize handlers
	wsHandler := gatewaywebsocket.NewEchoWebSocketHandler(gatewayUC, userUC)
	gatewayHandler := handler.NewHandler(gatewayUC, configs, nrApp, wsHandler)

	// Initialize NATS consumers
	if err := gatewayHandler.InitNATSConsumers(); err != nil {
		slogLogger.Error("Failed to initialize NATS consumers", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize Echo server
	e := echo.New()

	// Initialize enhanced health service
	healthService := health.NewHealthService(slogLogger)
	healthService.AddChecker("redis", health.NewRedisHealthChecker(redisClient))

	// Initialize middleware
	MW := middleware.NewMiddleware(configs, slogLogger, tracer)

	// Register enhanced health endpoints BEFORE applying middleware
	health.RegisterEnhancedHealthEndpoints(e, appName, Version, healthService)

	// Register additional health endpoint for /health/gateway
	healthGroup := e.Group("/health")
	healthGroup.GET("/gateway", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})

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

	// Shutdown New Relic
	if nrApp != nil {
		slogLogger.Info("Shutting down New Relic...")
		nrApp.Shutdown(10 * time.Second)
	}

	// Final log
	slogLogger.Info("Server exiting gracefully")
}
