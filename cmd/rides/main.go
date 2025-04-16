package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides/gateway"
	"github.com/piresc/nebengjek/services/rides/handler"
	"github.com/piresc/nebengjek/services/rides/repository"
	"github.com/piresc/nebengjek/services/rides/usecase"
)

func main() {
	appName := "rides-service"
	envPath := ".env"
	configs := config.InitConfig(envPath)

	// Initialize database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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
	rideRepo := repository.NewRideRepository(configs, postgresClient.GetDB())
	// Initialize NATS producer
	ridesGW := gateway.NewRideGW(natsClient.GetConn())
	// Initialize use cases
	rideUC, err := usecase.NewRideUC(configs, rideRepo, ridesGW)
	if err != nil {
		log.Fatalf("Failed to initialize ride use case: %v", err)
	}

	// Initialize Echo server
	e := echo.New()

	// Initialize handlers
	rideHandler := handler.NewRideHandler(configs, rideUC)

	// Initialize NATS consumers
	if err := rideHandler.InitNATSConsumers(); err != nil {
		log.Fatalf("Failed to initialize NATS consumers: %v", err)
	}

	// Start server
	serverAddr := fmt.Sprintf(":%d", configs.Server.Port)
	log.Printf("Starting %s on %s", appName, serverAddr)
	if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
