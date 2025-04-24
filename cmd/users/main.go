package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	wspkg "github.com/piresc/nebengjek/internal/pkg/websocket"
	"github.com/piresc/nebengjek/services/users/gateway"
	"github.com/piresc/nebengjek/services/users/handler"
	httpHandler "github.com/piresc/nebengjek/services/users/handler/http"
	natsHandler "github.com/piresc/nebengjek/services/users/handler/nats"
	wsHandler "github.com/piresc/nebengjek/services/users/handler/websocket"
	"github.com/piresc/nebengjek/services/users/repository"
	"github.com/piresc/nebengjek/services/users/usecase"
)

func main() {
	appName := "users-service"
	configs := config.InitConfig()

	// Initialize PostgreSQL database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresClient.Close()

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(configs.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize NATS
	natsClient, err := natspkg.NewClient(configs.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize repository
	userRepo := repository.NewUserRepo(configs, postgresClient.GetDB(), redisClient)

	// Initialize Gateway
	userGW := gateway.NewUserGW(natsClient)

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
		log.Fatalf("Failed to initialize NATS consumers: %v", err)
	}

	// Initialize handlers
	Handler := handler.NewHandler(userHandler, authHandler, wsManager, natsHandler, configs)

	// Initialize Echo router
	e := echo.New()
	Handler.RegisterRoutes(e)

	// Start server
	log.Printf("Starting %s on port %d", appName, configs.Server.Port)
	if err := e.Start(fmt.Sprintf(":%d", configs.Server.Port)); err != nil {
		log.Fatalf("Failed to start %s: %v", appName, err)
	}
}
