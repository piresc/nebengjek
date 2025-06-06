package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// MessageHandler is a function that processes NATS messages
type MessageHandler func(message []byte) error

// JetStreamMessageHandler is a function that processes JetStream messages with acknowledgment
type JetStreamMessageHandler func(msg jetstream.Msg) error

// Consumer handles consuming messages from NATS topics and JetStream
type Consumer struct {
	conn         *nats.Conn
	js           jetstream.JetStream
	subscription *nats.Subscription
	consumer     jetstream.Consumer
	consumeCtx   jetstream.ConsumeContext
	ctx          context.Context
	cancelFunc   context.CancelFunc
	isJetStream  bool
}

// NewConsumer creates a new NATS consumer for a topic/channel (legacy compatibility)
func NewConsumer(topic, queueGroup, address string, handler MessageHandler) (*Consumer, error) {
	// Connect to NATS server
	conn, err := nats.Connect(address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Subscribe to the topic with optional queue group
	var subscription *nats.Subscription
	if queueGroup != "" {
		subscription, err = conn.QueueSubscribe(topic, queueGroup, func(msg *nats.Msg) {
			// Process the message
			err := handler(msg.Data)
			if err != nil {
				logger.Debug("Error processing message",
					logger.String("topic", topic),
					logger.String("queue_group", queueGroup),
					logger.Err(err))
			}
		})
	} else {
		subscription, err = conn.Subscribe(topic, func(msg *nats.Msg) {
			// Process the message
			err := handler(msg.Data)
			if err != nil {
				logger.Debug("Error processing message",
					logger.String("topic", topic),
					logger.Err(err))
			}
		})
	}

	if err != nil {
		conn.Close()
		cancel()
		return nil, fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	return &Consumer{
		conn:         conn,
		subscription: subscription,
		ctx:          ctx,
		cancelFunc:   cancel,
		isJetStream:  false,
	}, nil
}

// NewJetStreamConsumer creates a new JetStream consumer with enhanced features
func NewJetStreamConsumer(client *Client, config ConsumerConfig, handler JetStreamMessageHandler) (*Consumer, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	// Create the consumer if it doesn't exist
	if err := client.CreateConsumer(config); err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Get the consumer
	consumerKey := fmt.Sprintf("%s:%s", config.StreamName, config.ConsumerName)
	consumer, exists := client.consumers[consumerKey]
	if !exists {
		return nil, fmt.Errorf("consumer %s not found after creation", consumerKey)
	}

	ctx, cancel := context.WithCancel(context.Background())

	jsConsumer := &Consumer{
		conn:        client.conn,
		js:          client.js,
		consumer:    consumer,
		ctx:         ctx,
		cancelFunc:  cancel,
		isJetStream: true,
	}

	// Start consuming messages
	if err := jsConsumer.startConsuming(handler); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	return jsConsumer, nil
}

// NewJetStreamPullConsumer creates a pull-based JetStream consumer
func NewJetStreamPullConsumer(client *Client, config ConsumerConfig) (*Consumer, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	// Ensure pull-based configuration
	config.AckPolicy = jetstream.AckExplicitPolicy
	config.DeliverPolicy = jetstream.DeliverAllPolicy

	// Create the consumer if it doesn't exist
	if err := client.CreateConsumer(config); err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Get the consumer
	consumerKey := fmt.Sprintf("%s:%s", config.StreamName, config.ConsumerName)
	consumer, exists := client.consumers[consumerKey]
	if !exists {
		return nil, fmt.Errorf("consumer %s not found after creation", consumerKey)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		conn:        client.conn,
		js:          client.js,
		consumer:    consumer,
		ctx:         ctx,
		cancelFunc:  cancel,
		isJetStream: true,
	}, nil
}

// startConsuming starts consuming messages with the provided handler
func (c *Consumer) startConsuming(handler JetStreamMessageHandler) error {
	if !c.isJetStream || c.consumer == nil {
		return fmt.Errorf("not a JetStream consumer")
	}

	consumeCtx, err := c.consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg); err != nil {
			logger.Error("Error processing JetStream message",
				logger.String("subject", msg.Subject()),
				logger.Err(err))

			// Negative acknowledgment for retry
			if nakErr := msg.Nak(); nakErr != nil {
				logger.Error("Failed to NAK message", logger.Err(nakErr))
			}
			return
		}

		// Acknowledge successful processing
		if ackErr := msg.Ack(); ackErr != nil {
			logger.Error("Failed to ACK message", logger.Err(ackErr))
		}
	})

	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.consumeCtx = consumeCtx

	// Stop consuming when context is cancelled
	go func() {
		<-c.ctx.Done()
		if c.consumeCtx != nil {
			c.consumeCtx.Stop()
		}
	}()

	return nil
}

// Fetch pulls messages manually (for pull consumers)
func (c *Consumer) Fetch(maxMessages int, timeout time.Duration) ([]jetstream.Msg, error) {
	if !c.isJetStream || c.consumer == nil {
		return nil, fmt.Errorf("not a JetStream consumer")
	}

	msgs, err := c.consumer.Fetch(maxMessages, jetstream.FetchMaxWait(timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	var result []jetstream.Msg
	for msg := range msgs.Messages() {
		result = append(result, msg)
	}

	if msgs.Error() != nil {
		return result, fmt.Errorf("error during fetch: %w", msgs.Error())
	}

	return result, nil
}

// FetchOne pulls a single message (for pull consumers)
func (c *Consumer) FetchOne(timeout time.Duration) (jetstream.Msg, error) {
	if !c.isJetStream || c.consumer == nil {
		return nil, fmt.Errorf("not a JetStream consumer")
	}

	msg, err := c.consumer.Next(jetstream.FetchMaxWait(timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	return msg, nil
}

// ProcessBatch processes a batch of messages with automatic acknowledgment
func (c *Consumer) ProcessBatch(maxMessages int, timeout time.Duration, handler JetStreamMessageHandler) error {
	msgs, err := c.Fetch(maxMessages, timeout)
	if err != nil {
		return fmt.Errorf("failed to fetch batch: %w", err)
	}

	for _, msg := range msgs {
		if err := handler(msg); err != nil {
			logger.Error("Error processing batch message",
				logger.String("subject", msg.Subject()),
				logger.Err(err))

			// Negative acknowledgment for retry
			if nakErr := msg.Nak(); nakErr != nil {
				logger.Error("Failed to NAK batch message", logger.Err(nakErr))
			}
			continue
		}

		// Acknowledge successful processing
		if ackErr := msg.Ack(); ackErr != nil {
			logger.Error("Failed to ACK batch message", logger.Err(ackErr))
		}
	}

	return nil
}

// GetInfo returns consumer information
func (c *Consumer) GetInfo() (*jetstream.ConsumerInfo, error) {
	if !c.isJetStream || c.consumer == nil {
		return nil, fmt.Errorf("not a JetStream consumer")
	}

	info, err := c.consumer.Info(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	return info, nil
}

// UpdateConsumer updates the consumer configuration
func (c *Consumer) UpdateConsumer(config jetstream.ConsumerConfig) error {
	if !c.isJetStream || c.consumer == nil {
		return fmt.Errorf("not a JetStream consumer")
	}

	// Get the stream first
	info, err := c.consumer.Info(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to get consumer info: %w", err)
	}

	stream, err := c.js.Stream(c.ctx, info.Stream)
	if err != nil {
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Update the consumer
	updatedConsumer, err := stream.UpdateConsumer(c.ctx, config)
	if err != nil {
		return fmt.Errorf("failed to update consumer: %w", err)
	}

	c.consumer = updatedConsumer
	logger.Info("Consumer updated successfully",
		logger.String("consumer", config.Name),
		logger.String("stream", info.Stream))

	return nil
}

// Pause pauses message consumption
func (c *Consumer) Pause() error {
	if c.consumeCtx != nil {
		c.consumeCtx.Stop()
		c.consumeCtx = nil
		logger.Info("Consumer paused")
	}
	return nil
}

// Resume resumes message consumption
func (c *Consumer) Resume(handler JetStreamMessageHandler) error {
	if !c.isJetStream || c.consumer == nil {
		return fmt.Errorf("not a JetStream consumer")
	}

	if c.consumeCtx != nil {
		return fmt.Errorf("consumer is already running")
	}

	return c.startConsuming(handler)
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	logger.Info("Stopping consumer")

	// Stop JetStream consume context
	if c.consumeCtx != nil {
		c.consumeCtx.Stop()
		c.consumeCtx = nil
	}

	// Unsubscribe from regular NATS subscription
	if c.subscription != nil {
		c.subscription.Unsubscribe()
		c.subscription = nil
	}

	// Cancel context
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	// Close connection if we own it (for legacy consumers)
	if !c.isJetStream && c.conn != nil {
		c.conn.Close()
	}
}

// IsActive returns true if the consumer is actively consuming messages
func (c *Consumer) IsActive() bool {
	if c.isJetStream {
		return c.consumeCtx != nil
	}
	return c.subscription != nil && c.subscription.IsValid()
}

// GetPendingMessages returns the number of pending messages
func (c *Consumer) GetPendingMessages() (uint64, error) {
	if !c.isJetStream || c.consumer == nil {
		return 0, fmt.Errorf("not a JetStream consumer")
	}

	info, err := c.consumer.Info(c.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get consumer info: %w", err)
	}

	return info.NumPending, nil
}

// GetAckPending returns the number of messages pending acknowledgment
func (c *Consumer) GetAckPending() (int, error) {
	if !c.isJetStream || c.consumer == nil {
		return 0, fmt.Errorf("not a JetStream consumer")
	}

	info, err := c.consumer.Info(c.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get consumer info: %w", err)
	}

	return info.NumAckPending, nil
}
