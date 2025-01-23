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
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type PaymentServer struct {
	producer     *nsq.Producer
	secretClient *secretmanager.Client
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
	consumer, err := nsq.NewConsumer("payment_events", "payment", config)
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
	paymentServer := &PaymentServer{producer: producer, secretClient: secretClient}
	proto.RegisterPaymentServiceServer(grpcServer, paymentServer)

	// Initialize REST server
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(nrgin.Middleware(app))

	// Setup REST routes
	setupRESTRoutes(router, paymentServer)

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
