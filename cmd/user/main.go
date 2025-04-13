package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/user/gateway"
	"github.com/piresc/nebengjek/services/user/handler"
	"github.com/piresc/nebengjek/services/user/repository"
	"github.com/piresc/nebengjek/services/user/usecase"
)

func main() {
	appName := "user-service"
	envPath := "./cmd/user/.env"
	configs := config.InitConfig(envPath)

	// Initialize PostgreSQL database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresClient.Close()

	// Initialize NATS
	natsClient, err := nats.NewClient(configs.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize repository, service, and handler
	userRepo := repository.NewUserRepo(configs, postgresClient.GetDB())
	userGW := gateway.NewUserGW(natsClient.GetConn())
	userUC := usecase.NewUserUC(*userRepo, *userGW)
	userHandler := handler.NewUserHandler(userUC)

	// Initialize Echo router
	e := echo.New()
	userHandler.RegisterRoutes(e)

	// Start server
	log.Printf("Starting %s on port %d", appName, configs.Server.Port)
	if err := e.Start(fmt.Sprintf(":%d", configs.Server.Port)); err != nil {
		log.Fatalf("Failed to start %s: %v", appName, err)
	}
}
