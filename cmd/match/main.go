package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match/gateway"
	"github.com/piresc/nebengjek/services/match/handler"
	"github.com/piresc/nebengjek/services/match/repository"
	"github.com/piresc/nebengjek/services/match/usecase"
)

func main() {
	appName := "match-service"
	envPath := "./cmd/match/.env"
	configs := config.InitConfig(envPath)

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
	natsClient, err := nats.NewClient(configs.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize repositories
	matchRepo := repository.NewMatchRepository(configs, postgresClient.GetDB(), redisClient)
	// Initialize gateways
	matchGW := gateway.NewMatchGW(natsClient.GetConn())
	// Initialize use case
	matchUC := usecase.NewMatchUC(matchRepo, matchGW)

	// Initialize Echo router and handler
	e := echo.New()
	matchHandler := handler.NewMatchHandler(matchUC, configs)

	// Initialize NATS consumers
	if err := matchHandler.InitNATSConsumers(); err != nil {
		log.Fatalf("Failed to initialize NATS consumers: %v", err)
	}

	// Start server
	log.Printf("Starting %s on port %d", appName, configs.Server.Port)
	if err := e.Start(fmt.Sprintf(":%d", configs.Server.Port)); err != nil {
		log.Fatalf("Failed to start %s: %v", appName, err)
	}
}
