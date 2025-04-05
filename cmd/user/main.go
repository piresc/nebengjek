package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/nebengjek/internal/pkg/config"
)

func main() {
	appName := "user-service"
	configs := config.InitConfig(appName)

	// Initialize the user service
	log.Printf("Starting %s on port %d", appName, configs.Server.Port)

	// TODO: Initialize your user service handlers here
	// Example: router := initializeRouter()

	if err := http.ListenAndServe(fmt.Sprintf(":%d", configs.Server.Port), nil); err != nil {
		log.Fatalf("Failed to start %s: %v", appName, err)
	}
}
