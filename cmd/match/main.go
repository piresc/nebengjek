package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/services/match/handler"
	"github.com/piresc/nebengjek/services/match/repository"
	"github.com/piresc/nebengjek/services/match/usecase"
)

func main() {
	appName := "match-service"
	configs := config.InitConfig(appName)

	// Initialize PostgreSQL database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresClient.Close()

	// Initialize repositories
	matchRepo := repository.NewMatchRepository(configs, postgresClient.GetDB())
	// Initialize use case
	matchUC := usecase.NewMatchUseCase(configs, matchRepo)

	// Initialize NATS consumers
	err = matchUC.InitConsumers()
	if err != nil {
		log.Fatalf("Failed to initialize NATS consumers: %v", err)
	}

	// Initialize Echo router and handler
	e := echo.New()
	matchHandler := handler.NewMatchHandler(matchUC)
	matchHandler.RegisterRoutes(e)

	// Start server
	log.Printf("Starting %s on port %d", appName, configs.Server.Port)
	if err := e.Start(fmt.Sprintf(":%d", configs.Server.Port)); err != nil {
		log.Fatalf("Failed to start %s: %v", appName, err)
	}
}
