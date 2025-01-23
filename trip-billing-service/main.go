package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/nsqio/go-nsq"
	"github.com/piresc/nebengjek/trip-billing-service/proto"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file: %s", err)
	}
}

func main() {
	// Initialize Secret Manager
	ctx := context.Background()
	secretClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatal("Failed to create Secret Manager client: ", err)
	}
	defer secretClient.Close()

	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(viper.GetString("newrelic.app_name")),
		newrelic.ConfigLicense(viper.GetString("newrelic.license_key")),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize NSQ Consumer (for async payment status)
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer("trip_events", "billing", config)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Stop()

	// Initialize NSQ Producer (for async notifications)
	producer, err := nsq.NewProducer(viper.GetString("nsq.address"), config)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Stop()

	// Initialize gRPC server
	lis, err := net.Listen("tcp", viper.GetString("server.address"))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	billingServer := &BillingServer{producer: producer, secretClient: secretClient}
	proto.RegisterTripBillingServiceServer(grpcServer, billingServer)

	// Initialize REST server
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(nrgin.Middleware(app))

	// Setup REST routes
	setupRESTRoutes(router, billingServer)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start gRPC server
	go func() {
		log.Printf("Starting gRPC server on %s", viper.GetString("server.address"))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	// Start REST server
	go func() {
		log.Printf("Starting REST server on %s", viper.GetString("server.rest_port"))
		if err := router.Run(viper.GetString("server.rest_port")); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")
	grpcServer.GracefulStop()
}

func setupRoutes(router *gin.Engine, producer *nsq.Producer) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Trip and Billing routes
	trip := router.Group("/trip")
	{
		trip.POST("/start", handleTripStart(producer))
		trip.POST("/end", handleTripEnd(producer))
		trip.GET("/cost/:id", handleTripCost())
		trip.GET("/history/:user_id", handleTripHistory())
	}
}
