package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// StreamConfig defines configuration for a JetStream stream
type StreamConfig struct {
	Name      string
	Subjects  []string
	Retention jetstream.RetentionPolicy
	Storage   jetstream.StorageType
	Replicas  int
	MaxAge    time.Duration
	MaxBytes  int64
	MaxMsgs   int64
	Discard   jetstream.DiscardPolicy
}

// ConsumerConfig defines configuration for a JetStream consumer
type ConsumerConfig struct {
	StreamName    string
	ConsumerName  string
	Subject       string
	DeliverPolicy jetstream.DeliverPolicy
	AckPolicy     jetstream.AckPolicy
	AckWait       time.Duration
	MaxDeliver    int
	FilterSubject string
	ReplayPolicy  jetstream.ReplayPolicy
	RateLimitBps  uint64
	MaxAckPending int
}

// PublishOptions defines options for publishing messages
type PublishOptions struct {
	Subject     string
	Data        []byte
	Headers     nats.Header
	MsgID       string
	ExpectedSeq uint64
	Timeout     time.Duration
}

// Client represents a JetStream-enabled NATS client
type Client struct {
	conn       *nats.Conn
	js         jetstream.JetStream
	ctx        context.Context
	streams    map[string]jetstream.Stream
	consumers  map[string]jetstream.Consumer
	cancelFunc context.CancelFunc
}

// NewClient creates a new JetStream-enabled NATS client
func NewClient(url string) (*Client, error) {
	// Connect to NATS server with JetStream options
	opts := []nats.Option{
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.PingInterval(30 * time.Second),
		nats.MaxPingsOutstanding(3),
		nats.ReconnectBufSize(5 * 1024 * 1024), // 5MB buffer
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Error("NATS disconnected", logger.Err(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", logger.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Info("NATS connection closed")
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		conn:       conn,
		js:         js,
		ctx:        ctx,
		streams:    make(map[string]jetstream.Stream),
		consumers:  make(map[string]jetstream.Consumer),
		cancelFunc: cancel,
	}

	// Initialize default streams for the ride-sharing system
	if err := client.initializeDefaultStreams(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize default streams: %w", err)
	}

	return client, nil
}

// initializeDefaultStreams creates the default streams for the ride-sharing system
func (c *Client) initializeDefaultStreams() error {
	// Use the centralized stream configurations that support dual consumption
	defaultStreams := DefaultStreamConfigs()

	for _, streamConfig := range defaultStreams {
		if err := c.CreateOrUpdateStream(streamConfig); err != nil {
			return fmt.Errorf("failed to create stream %s: %w", streamConfig.Name, err)
		}
	}

	return nil
}

// CreateOrUpdateStream creates or updates a JetStream stream
func (c *Client) CreateOrUpdateStream(config StreamConfig) error {
	streamConfig := jetstream.StreamConfig{
		Name:       config.Name,
		Subjects:   config.Subjects,
		Retention:  config.Retention,
		Storage:    config.Storage,
		Replicas:   config.Replicas,
		MaxAge:     config.MaxAge,
		MaxBytes:   config.MaxBytes,
		MaxMsgs:    config.MaxMsgs,
		Discard:    config.Discard,
		NoAck:      false,
		Duplicates: 5 * time.Minute, // Duplicate detection window
	}

	stream, err := c.js.CreateOrUpdateStream(c.ctx, streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create/update stream: %w", err)
	}

	c.streams[config.Name] = stream
	logger.Info("Stream created/updated successfully",
		logger.String("stream", config.Name),
		logger.Strings("subjects", config.Subjects))

	return nil
}

// CreateConsumer creates a durable consumer for a stream
func (c *Client) CreateConsumer(config ConsumerConfig) error {
	stream, exists := c.streams[config.StreamName]
	if !exists {
		return fmt.Errorf("stream %s not found", config.StreamName)
	}

	consumerConfig := jetstream.ConsumerConfig{
		Name:          config.ConsumerName,
		DeliverPolicy: config.DeliverPolicy,
		AckPolicy:     config.AckPolicy,
		AckWait:       config.AckWait,
		MaxDeliver:    config.MaxDeliver,
		FilterSubject: config.FilterSubject,
		ReplayPolicy:  config.ReplayPolicy,
		RateLimit:     config.RateLimitBps,
		MaxAckPending: config.MaxAckPending,
	}

	consumer, err := stream.CreateOrUpdateConsumer(c.ctx, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	consumerKey := fmt.Sprintf("%s:%s", config.StreamName, config.ConsumerName)
	c.consumers[consumerKey] = consumer

	logger.Info("Consumer created successfully",
		logger.String("stream", config.StreamName),
		logger.String("consumer", config.ConsumerName),
		logger.String("subject", config.FilterSubject),
		logger.String("deliver_policy", fmt.Sprintf("%v", config.DeliverPolicy)))

	return nil
}

// RecreateConsumer deletes and recreates a consumer with new configuration
func (c *Client) RecreateConsumer(config ConsumerConfig) error {
	stream, exists := c.streams[config.StreamName]
	if !exists {
		return fmt.Errorf("stream %s not found", config.StreamName)
	}

	// Try to delete existing consumer (ignore error if it doesn't exist)
	if err := stream.DeleteConsumer(c.ctx, config.ConsumerName); err != nil {
		logger.Info("Consumer not found for deletion (this is expected for new consumers)",
			logger.String("consumer", config.ConsumerName),
			logger.String("stream", config.StreamName))
	} else {
		logger.Info("Deleted existing consumer for recreation",
			logger.String("consumer", config.ConsumerName),
			logger.String("stream", config.StreamName))
	}

	// Remove from local cache
	consumerKey := fmt.Sprintf("%s:%s", config.StreamName, config.ConsumerName)
	delete(c.consumers, consumerKey)

	// Create new consumer with updated configuration
	return c.CreateConsumer(config)
}

// Publish publishes a message to JetStream with delivery guarantees
func (c *Client) Publish(subject string, data []byte) error {
	opts := PublishOptions{
		Subject: subject,
		Data:    data,
		Timeout: 10 * time.Second,
	}
	return c.PublishWithOptions(opts)
}

// PublishWithOptions publishes a message with custom options
func (c *Client) PublishWithOptions(opts PublishOptions) error {
	ctx := c.ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(c.ctx, opts.Timeout)
		defer cancel()
	}

	pubOpts := []jetstream.PublishOpt{}

	if opts.MsgID != "" {
		pubOpts = append(pubOpts, jetstream.WithMsgID(opts.MsgID))
	}

	if opts.ExpectedSeq > 0 {
		pubOpts = append(pubOpts, jetstream.WithExpectLastSequence(opts.ExpectedSeq))
	}

	msg := &nats.Msg{
		Subject: opts.Subject,
		Data:    opts.Data,
		Header:  opts.Headers,
	}

	ack, err := c.js.PublishMsg(ctx, msg, pubOpts...)
	if err != nil {
		return fmt.Errorf("failed to publish message to subject %s: %w", opts.Subject, err)
	}

	logger.Debug("Message published successfully",
		logger.String("subject", opts.Subject),
		logger.String("stream", ack.Stream),
		logger.Int64("sequence", int64(ack.Sequence)))

	return nil
}

// PublishAsync publishes a message asynchronously
func (c *Client) PublishAsync(subject string, data []byte, handler func(*jetstream.PubAck, error)) error {
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
	}

	future, err := c.js.PublishMsgAsync(msg)
	if err != nil {
		return fmt.Errorf("failed to publish async message: %w", err)
	}

	// Handle the result asynchronously
	go func() {
		select {
		case ack := <-future.Ok():
			handler(ack, nil)
		case err := <-future.Err():
			handler(nil, err)
		case <-c.ctx.Done():
			handler(nil, c.ctx.Err())
		}
	}()

	return nil
}

// Subscribe creates a subscription with automatic acknowledgment
func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	// For backward compatibility, create a simple subscription
	sub, err := c.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	logger.Info("Subscribed to subject", logger.String("subject", subject))
	return sub, nil
}

// ConsumeMessages consumes messages from a JetStream consumer
func (c *Client) ConsumeMessages(streamName, consumerName string, handler func(jetstream.Msg) error) error {
	consumerKey := fmt.Sprintf("%s:%s", streamName, consumerName)
	consumer, exists := c.consumers[consumerKey]
	if !exists {
		return fmt.Errorf("consumer %s not found", consumerKey)
	}

	// Create a consume context
	consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg); err != nil {
			logger.Error("Error processing message",
				logger.String("consumer", consumerKey),
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

	logger.Info("Started consuming messages",
		logger.String("stream", streamName),
		logger.String("consumer", consumerName))

	// Keep consuming until context is cancelled
	go func() {
		<-c.ctx.Done()
		consumeCtx.Stop()
	}()

	return nil
}

// Request sends a request and waits for a response (maintained for compatibility)
func (c *Client) Request(subject string, data []byte) (*nats.Msg, error) {
	msg, err := c.conn.Request(subject, data, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to subject %s: %w", subject, err)
	}
	return msg, nil
}

// GetConn returns the underlying NATS connection (maintained for compatibility)
func (c *Client) GetConn() *nats.Conn {
	return c.conn
}

// GetJetStream returns the JetStream context
func (c *Client) GetJetStream() jetstream.JetStream {
	return c.js
}

// GetStream returns a specific stream
func (c *Client) GetStream(name string) (jetstream.Stream, error) {
	stream, exists := c.streams[name]
	if !exists {
		return nil, fmt.Errorf("stream %s not found", name)
	}
	return stream, nil
}

// GetConsumer returns a specific consumer
func (c *Client) GetConsumer(streamName, consumerName string) (jetstream.Consumer, error) {
	consumerKey := fmt.Sprintf("%s:%s", streamName, consumerName)
	consumer, exists := c.consumers[consumerKey]
	if !exists {
		return nil, fmt.Errorf("consumer %s not found", consumerKey)
	}
	return consumer, nil
}

// ListStreams returns information about all streams
func (c *Client) ListStreams() ([]jetstream.StreamInfo, error) {
	var streams []jetstream.StreamInfo

	streamInfos := c.js.ListStreams(c.ctx)
	for info := range streamInfos.Info() {
		streams = append(streams, *info)
	}

	if streamInfos.Err() != nil {
		return nil, fmt.Errorf("failed to list streams: %w", streamInfos.Err())
	}

	return streams, nil
}

// DeleteStream deletes a stream and all its consumers
func (c *Client) DeleteStream(name string) error {
	_, exists := c.streams[name]
	if !exists {
		return fmt.Errorf("stream %s not found", name)
	}

	if err := c.js.DeleteStream(c.ctx, name); err != nil {
		return fmt.Errorf("failed to delete stream %s: %w", name, err)
	}

	delete(c.streams, name)

	// Remove associated consumers
	for key := range c.consumers {
		if fmt.Sprintf("%s:", name) == key[:len(name)+1] {
			delete(c.consumers, key)
		}
	}

	logger.Info("Stream deleted successfully", logger.String("stream", name))
	return nil
}

// PurgeStream purges all messages from a stream
func (c *Client) PurgeStream(name string) error {
	stream, exists := c.streams[name]
	if !exists {
		return fmt.Errorf("stream %s not found", name)
	}

	if err := stream.Purge(c.ctx); err != nil {
		return fmt.Errorf("failed to purge stream %s: %w", name, err)
	}

	logger.Info("Stream purged successfully", logger.String("stream", name))
	return nil
}

// GetStreamInfo returns information about a specific stream
func (c *Client) GetStreamInfo(name string) (*jetstream.StreamInfo, error) {
	stream, exists := c.streams[name]
	if !exists {
		return nil, fmt.Errorf("stream %s not found", name)
	}

	info, err := stream.Info(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	return info, nil
}

// Close closes the JetStream client and all connections
func (c *Client) Close() {
	logger.Info("Closing JetStream client")

	// Cancel context to stop all consumers
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	// Close NATS connection
	if c.conn != nil {
		c.conn.Close()
	}

	// Clear maps
	c.streams = make(map[string]jetstream.Stream)
	c.consumers = make(map[string]jetstream.Consumer)
}

// IsConnected returns true if the client is connected to NATS
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Stats returns connection statistics
func (c *Client) Stats() nats.Statistics {
	if c.conn == nil {
		return nats.Statistics{}
	}
	return c.conn.Stats()
}
