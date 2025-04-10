package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/services/rides/handler"
	"github.com/piresc/nebengjek/services/rides/repository"
	"github.com/piresc/nebengjek/services/rides/usecase"
)

func main() {
	appName := "rides-service"
	cfg := config.InitConfig(appName)

	// Initialize database connection
	postgresClient, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer postgresClient.Close()

	// Initialize Redis client
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	rideRepo := repository.NewRideRepository(cfg, postgresClient.GetDB())

	// Initialize use cases
	rideUC, err := usecase.NewRideUC(cfg, rideRepo)
	if err != nil {
		log.Fatalf("Failed to initialize ride use case: %v", err)
	}

	// Initialize handlers
	rideHandler := handler.NewRideHandler(rideUC)

	// Initialize Echo server
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Register routes
	rideHandler.RegisterRoutes(e)

	// Start server
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting %s on %s", appName, serverAddr)
	if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
