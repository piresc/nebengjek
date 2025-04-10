package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/services/billing/handler"
	"github.com/piresc/nebengjek/services/billing/repository"
	"github.com/piresc/nebengjek/services/billing/usecase"
)

func main() {
	appName := "billing-service"
	configs := config.InitConfig(appName)

	// Initialize PostgreSQL database connection
	postgresClient, err := database.NewPostgresClient(configs.Database)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresClient.Close()

	// Initialize NATS connection
	natsClient, err := nats.Connect(configs.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize repository, service, and handler
	billingRepo := repository.NewBillingRepository(postgresClient.GetDB())
	locationRepo := repository.NewLocationRepository()
	billingUC := usecase.NewBillingUC(configs, billingRepo, locationRepo, natsClient)
	billingHandler := handler.NewBillingHandler(billingUC)

	// Initialize Echo router
	e := echo.New()
	billingHandler.RegisterRoutes(e)

	// Start server
	log.Printf("Starting %s on port %d", appName, configs.Server.Port)
	if err := e.Start(fmt.Sprintf(":%d", configs.Server.Port)); err != nil {
		log.Fatalf("Failed to start %s: %v", appName, err)
	}
}
