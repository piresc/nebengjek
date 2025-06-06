package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// ExampleUsage demonstrates how to use the new JetStream client
func ExampleUsage() {
	// Create a new JetStream client
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		logger.Error("Failed to create client", logger.Err(err))
		return
	}
	defer client.Close()

	// Example 1: Publishing messages with delivery guarantees
	err = client.Publish("user.beacon", []byte(`{"user_id": "123", "location": {"lat": 37.7749, "lng": -122.4194}}`))
	if err != nil {
		logger.Error("Failed to publish message", logger.Err(err))
		return
	}

	// Example 2: Publishing with custom options
	err = client.PublishWithOptions(PublishOptions{
		Subject: "match.found",
		Data:    []byte(`{"match_id": "456", "driver_id": "789", "passenger_id": "123"}`),
		MsgID:   "match-456",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		logger.Error("Failed to publish with options", logger.Err(err))
		return
	}

	// Example 3: Creating a custom consumer
	consumerConfig := NewConsumerConfigBuilder("MATCH_STREAM", "custom_match_consumer").
		WithSubject("match.found").
		WithDeliverPolicy(jetstream.DeliverAllPolicy).
		WithAckPolicy(jetstream.AckExplicitPolicy).
		WithMaxDeliver(3).
		WithAckWait(30 * time.Second).
		Build()

	consumer, err := NewJetStreamConsumer(client, consumerConfig, func(msg jetstream.Msg) error {
		logger.Info("Received match found message",
			logger.String("subject", msg.Subject()),
			logger.String("data", string(msg.Data())))

		// Process the message here
		// Return error if processing fails (message will be retried)
		return nil
	})
	if err != nil {
		logger.Error("Failed to create consumer", logger.Err(err))
		return
	}
	defer consumer.Stop()

	// Example 4: Pull-based consumer for batch processing
	pullConsumer, err := NewJetStreamPullConsumer(client, consumerConfig)
	if err != nil {
		logger.Error("Failed to create pull consumer", logger.Err(err))
		return
	}
	defer pullConsumer.Stop()

	// Process messages in batches
	err = pullConsumer.ProcessBatch(10, 5*time.Second, func(msg jetstream.Msg) error {
		logger.Info("Processing batch message",
			logger.String("subject", msg.Subject()),
			logger.String("data", string(msg.Data())))
		return nil
	})
	if err != nil {
		logger.Error("Failed to process batch", logger.Err(err))
	}
}

// ExampleServiceIntegration shows how to integrate JetStream into a service
func ExampleServiceIntegration() {
	// Initialize client
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		logger.Error("Failed to create client", logger.Err(err))
		return
	}
	defer client.Close()

	// Create default consumers for the users service
	err = CreateDefaultConsumersForService(client, "users")
	if err != nil {
		logger.Error("Failed to create default consumers", logger.Err(err))
		return
	}

	// Start consuming match found events
	matchConsumer, err := NewJetStreamConsumer(client,
		DefaultConsumerConfigs()["match_found_consumer"],
		handleMatchFoundEvent)
	if err != nil {
		logger.Error("Failed to create match consumer", logger.Err(err))
		return
	}
	defer matchConsumer.Stop()

	// Start consuming ride events
	rideConsumer, err := NewJetStreamConsumer(client,
		DefaultConsumerConfigs()["ride_pickup_consumer"],
		handleRidePickupEvent)
	if err != nil {
		logger.Error("Failed to create ride consumer", logger.Err(err))
		return
	}
	defer rideConsumer.Stop()

	// Keep the service running
	select {}
}

// handleMatchFoundEvent processes match found events
func handleMatchFoundEvent(msg jetstream.Msg) error {
	logger.Info("Processing match found event",
		logger.String("subject", msg.Subject()),
		logger.String("data", string(msg.Data())))

	// Add your business logic here
	// For example: notify users via WebSocket, update database, etc.

	return nil
}

// handleRidePickupEvent processes ride pickup events
func handleRidePickupEvent(msg jetstream.Msg) error {
	logger.Info("Processing ride pickup event",
		logger.String("subject", msg.Subject()),
		logger.String("data", string(msg.Data())))

	// Add your business logic here
	// For example: update ride status, notify driver, etc.

	return nil
}

// ExampleStreamManagement demonstrates stream management operations
func ExampleStreamManagement() {
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		logger.Error("Failed to create client", logger.Err(err))
		return
	}
	defer client.Close()

	// Create a custom stream
	customStream := NewStreamConfigBuilder("CUSTOM_STREAM").
		WithSubjects("custom.events", "custom.notifications").
		WithRetention(jetstream.WorkQueuePolicy).
		WithStorage(jetstream.FileStorage).
		WithMaxAge(1 * time.Hour).
		WithMaxBytes(50 * 1024 * 1024).
		Build()

	err = client.CreateOrUpdateStream(customStream)
	if err != nil {
		logger.Error("Failed to create custom stream", logger.Err(err))
		return
	}

	// List all streams
	streams, err := client.ListStreams()
	if err != nil {
		logger.Error("Failed to list streams", logger.Err(err))
		return
	}

	for _, stream := range streams {
		logger.Info("Stream info",
			logger.String("name", stream.Config.Name),
			logger.Int64("messages", int64(stream.State.Msgs)),
			logger.Int64("bytes", int64(stream.State.Bytes)))
	}

	// Get specific stream info
	streamInfo, err := client.GetStreamInfo("USER_STREAM")
	if err != nil {
		logger.Error("Failed to get stream info", logger.Err(err))
		return
	}

	logger.Info("USER_STREAM details",
		logger.Int64("messages", int64(streamInfo.State.Msgs)),
		logger.Int64("bytes", int64(streamInfo.State.Bytes)),
		logger.Int("consumers", streamInfo.State.Consumers))

	// Purge a stream (remove all messages)
	err = client.PurgeStream("CUSTOM_STREAM")
	if err != nil {
		logger.Error("Failed to purge stream", logger.Err(err))
		return
	}

	// Delete a stream
	err = client.DeleteStream("CUSTOM_STREAM")
	if err != nil {
		logger.Error("Failed to delete stream", logger.Err(err))
		return
	}
}

// ExampleErrorHandling demonstrates error handling and retry mechanisms
func ExampleErrorHandling() {
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		logger.Error("Failed to create client", logger.Err(err))
		return
	}
	defer client.Close()

	// Consumer with error handling and retry logic
	consumerConfig := NewConsumerConfigBuilder("MATCH_STREAM", "error_handling_consumer").
		WithSubject("match.found").
		WithDeliverPolicy(jetstream.DeliverAllPolicy).
		WithAckPolicy(jetstream.AckExplicitPolicy).
		WithMaxDeliver(5). // Retry up to 5 times
		WithAckWait(30 * time.Second).
		Build()

	consumer, err := NewJetStreamConsumer(client, consumerConfig, func(msg jetstream.Msg) error {
		// Simulate processing that might fail
		if string(msg.Data()) == "bad_data" {
			logger.Error("Failed to process message",
				logger.String("data", string(msg.Data())))
			return fmt.Errorf("invalid data format")
		}

		// Successful processing
		logger.Info("Successfully processed message",
			logger.String("data", string(msg.Data())))
		return nil
	})
	if err != nil {
		logger.Error("Failed to create consumer", logger.Err(err))
		return
	}
	defer consumer.Stop()

	// Publish test messages
	client.Publish("match.found", []byte("good_data"))
	client.Publish("match.found", []byte("bad_data")) // This will be retried

	// Monitor consumer status
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if !consumer.IsActive() {
				logger.Warn("Consumer is not active")
				continue
			}

			pending, err := consumer.GetPendingMessages()
			if err != nil {
				logger.Error("Failed to get pending messages", logger.Err(err))
				continue
			}

			ackPending, err := consumer.GetAckPending()
			if err != nil {
				logger.Error("Failed to get ack pending", logger.Err(err))
				continue
			}

			logger.Info("Consumer status",
				logger.Int64("pending", int64(pending)),
				logger.Int("ack_pending", ackPending))
		}
	}()

	// Keep running for demonstration
	time.Sleep(1 * time.Minute)
}

// ExampleAsyncPublishing demonstrates asynchronous publishing
func ExampleAsyncPublishing() {
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		logger.Error("Failed to create client", logger.Err(err))
		return
	}
	defer client.Close()

	// Publish messages asynchronously
	for i := 0; i < 100; i++ {
		data := fmt.Sprintf(`{"event_id": %d, "timestamp": "%s"}`, i, time.Now().Format(time.RFC3339))

		err = client.PublishAsync("user.beacon", []byte(data), func(ack *jetstream.PubAck, err error) {
			if err != nil {
				logger.Error("Failed to publish async message", logger.Err(err))
				return
			}

			logger.Debug("Async message published",
				logger.String("stream", ack.Stream),
				logger.Int64("sequence", int64(ack.Sequence)))
		})

		if err != nil {
			logger.Error("Failed to initiate async publish", logger.Err(err))
		}
	}

	// Wait for async operations to complete
	time.Sleep(5 * time.Second)
}

// ExampleBackwardCompatibility shows how existing code can work with minimal changes
func ExampleBackwardCompatibility() {
	// This works exactly like the old NATS client
	client, err := NewClient("nats://localhost:4222")
	if err != nil {
		logger.Error("Failed to create client", logger.Err(err))
		return
	}
	defer client.Close()

	// Old-style publishing (still works)
	err = client.Publish("user.beacon", []byte(`{"user_id": "123"}`))
	if err != nil {
		logger.Error("Failed to publish", logger.Err(err))
		return
	}

	// Old-style subscription (still works)
	sub, err := client.Subscribe("user.beacon", func(msg *nats.Msg) {
		logger.Info("Received message", logger.String("data", string(msg.Data)))
	})
	if err != nil {
		logger.Error("Failed to subscribe", logger.Err(err))
		return
	}
	defer sub.Unsubscribe()

	// Old-style request-response (still works)
	response, err := client.Request("user.status", []byte(`{"user_id": "123"}`))
	if err != nil {
		logger.Error("Failed to send request", logger.Err(err))
		return
	}

	logger.Info("Received response", logger.String("data", string(response.Data)))

	// Access to underlying connection (still works)
	conn := client.GetConn()
	logger.Info("Connection status", logger.Bool("connected", conn.IsConnected()))

	// New JetStream features are also available
	js := client.GetJetStream()
	logger.Info("JetStream available", logger.Bool("available", js != nil))
}
