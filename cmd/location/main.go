package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location/gateway"
	"github.com/piresc/nebengjek/services/location/handler"
	"github.com/piresc/nebengjek/services/location/repository"
	"github.com/piresc/nebengjek/services/location/usecase"
)

func main() {
	appName := "location-service"
	configs := config.InitConfig()

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(configs.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize NATS client
	natsClient, err := nats.NewClient(configs.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize repository
	locationRepo := repository.NewLocationRepository(redisClient)

	// Initialize gateway
	locationGW := gateway.NewLocationGW(natsClient)

	// Initialize usecase
	locationUC := usecase.NewLocationUC(locationRepo, locationGW)

	// Initialize NATS handler
	locationhandler := handler.NewLocationHandler(locationUC, natsClient)

	// Initialize NATS consumers
	if err := locationhandler.InitNATSConsumers(); err != nil {
		log.Fatalf("Failed to initialize locationhandler consumers: %v", err)
	}

	// Initialize Echo router
	e := echo.New()

	// Start server
	log.Printf("Starting %s on port %d", appName, configs.Server.Port)
	if err := e.Start(fmt.Sprintf(":%d", configs.Server.Port)); err != nil {
		log.Fatalf("Failed to start %s: %v", appName, err)
	}
}
