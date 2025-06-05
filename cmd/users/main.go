package main

import (
	"fmt"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/health"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/middleware"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	wspkg "github.com/piresc/nebengjek/internal/pkg/websocket"
	"github.com/piresc/nebengjek/services/users/gateway"
	"github.com/piresc/nebengjek/services/users/handler"
	httpHandler "github.com/piresc/nebengjek/services/users/handler/http"
	natsHandler "github.com/piresc/nebengjek/services/users/handler/nats"
	wsHandler "github.com/piresc/nebengjek/services/users/handler/websocket"
	"github.com/piresc/nebengjek/services/users/repository"
	"github.com/piresc/nebengjek/services/users/usecase"
	"go.uber.org/zap"
)

func main() {
	appName := "users-service"
	configPath := "/Users/pirescerullo/GitHub/assessment/nebengjek/config/users.env"
	configs := config.InitConfig(configPath)

	// Initialize New Relic and Zap logger
	nrApp := nrpkg.InitNewRelic(configs)

	// Wait for New Relic connection before proceeding
	if nrApp != nil {
		if err := nrApp.WaitForConnection(10 * time.Second); err != nil {
			log.Printf("Warning: New Relic connection timeout: %v", err)
		} else {
			log.Println("New Relic connection established")
		}
	}

	zapLogger, err := logger.InitZapLoggerFromConfig(configs, nrApp)
	if err != nil {
		log.Fatalf("Failed to create Zap logger: %v", err)
	}
	defer zapLogger.Close()

	// Log startup with Zap
	zapLogger.Info("Starting application",
		zap.String("app", appName),
		zap.String("version", configs.App.Version),
		zap.String("environment", configs.App.Environment),
	)

	// Initialize PostgreSQL database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		zapLogger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer postgresClient.Close()

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(configs.Redis)
	if err != nil {
		zapLogger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Initialize NATS
	natsClient, err := natspkg.NewClient(configs.NATS.URL)
	if err != nil {
		zapLogger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer natsClient.Close()

	// Initialize repository
	userRepo := repository.NewUserRepo(configs, postgresClient.GetDB(), redisClient)

	// Initialize Gateway
	userGW := gateway.NewUserGW(natsClient, configs.Services.MatchServiceURL, configs.Services.RidesServiceURL)

	// Initialize UseCase
	userUC := usecase.NewUserUC(userRepo, userGW, configs)

	// Handlers for HTTP
	userHandler := httpHandler.NewUserHandler(userUC)
	authHandler := httpHandler.NewAuthHandler(userUC)

	// Handlers for WebSocket
	manager := wspkg.NewManager(configs.JWT)
	wsManager := wsHandler.NewWebSocketManager(userUC, manager)

	// Handlers for NATS
	natsHandler := natsHandler.NewNatsHandler(wsManager, natsClient)

	// Initialize NATS consumers
	if err := natsHandler.InitConsumers(); err != nil {
		zapLogger.Fatal("Failed to initialize NATS consumers", zap.Error(err))
	}

	// Initialize handlers
	Handler := handler.NewHandler(userHandler, authHandler, wsManager, natsHandler, configs)

	// Initialize Echo router
	e := echo.New()

	// Add middlewares
	e.Use(middleware.RequestIDMiddleware())
	e.Use(logger.ZapEchoMiddleware(zapLogger))

	// Register health endpoints
	health.RegisterHealthEndpoints(e, appName)

	// Register service routes
	Handler.RegisterRoutes(e)

	// Start server
	zapLogger.Info("Starting server",
		zap.String("app", appName),
		zap.Int("port", configs.Server.Port),
	)

	if err := e.Start(fmt.Sprintf(":%d", configs.Server.Port)); err != nil {
		zapLogger.Fatal("Failed to start server",
			zap.String("app", appName),
			zap.Error(err),
		)
	}
}
