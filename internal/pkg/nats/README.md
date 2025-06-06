# JetStream NATS Client Implementation

This package provides a comprehensive JetStream-enabled NATS client implementation for the nebengjek ride-sharing system. It replaces the basic NATS client with advanced features including message persistence, delivery guarantees, and stream management.

## Overview

The implementation includes:
- **JetStream Client**: Enhanced NATS client with JetStream capabilities
- **Stream Management**: Automatic creation and management of message streams
- **Consumer Management**: Durable consumers with acknowledgment and retry logic
- **Backward Compatibility**: Existing code continues to work with minimal changes
- **Helper Functions**: Builder patterns and utilities for easy configuration

## Key Features

### 🚀 JetStream Capabilities
- **At-least-once delivery** guarantees
- **Message persistence** with configurable retention policies
- **Automatic retry** with exponential backoff
- **Dead letter queue** handling
- **Stream replay** capabilities
- **Consumer acknowledgment** with timeout handling

### 🏗️ Stream Architecture

The system uses four main streams optimized for the ride-sharing domain:

#### USER_STREAM
- **Subjects**: `user.beacon`, `user.finder`
- **Retention**: Interest-based (messages deleted when all consumers acknowledge)
- **Storage**: File storage for durability
- **Max Age**: 24 hours
- **Use Case**: User location beacons and ride finder requests

#### MATCH_STREAM
- **Subjects**: `match.found`, `match.rejected`, `match.accepted`
- **Retention**: Work queue (messages deleted after acknowledgment)
- **Storage**: File storage
- **Max Age**: 1 hour
- **Use Case**: Driver-passenger matching events

#### RIDE_STREAM
- **Subjects**: `ride.pickup`, `ride.started`, `ride.arrived`, `ride.completed`
- **Retention**: Limits-based (messages kept for audit purposes)
- **Storage**: File storage
- **Max Age**: 7 days
- **Use Case**: Ride lifecycle events and audit trail

#### LOCATION_STREAM
- **Subjects**: `location.update`, `location.aggregate`
- **Retention**: Interest-based
- **Storage**: Memory storage for fast access
- **Max Age**: 2 hours
- **Use Case**: Real-time location updates and aggregation

## Quick Start

### Basic Usage

```go
// Create a new JetStream client
client, err := nats.NewClient("nats://localhost:4222")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Publish with delivery guarantees
err = client.Publish("user.beacon", []byte(`{"user_id": "123", "lat": 37.7749, "lng": -122.4194}`))
if err != nil {
    log.Fatal(err)
}
```

### Advanced Publishing

```go
// Publish with custom options
err = client.PublishWithOptions(nats.PublishOptions{
    Subject: "match.found",
    Data:    []byte(`{"match_id": "456", "driver_id": "789"}`),
    MsgID:   "match-456", // Deduplication ID
    Timeout: 5 * time.Second,
})

// Async publishing with callback
err = client.PublishAsync("ride.started", data, func(ack *jetstream.PubAck, err error) {
    if err != nil {
        log.Printf("Publish failed: %v", err)
        return
    }
    log.Printf("Message published to stream %s, sequence %d", ack.Stream, ack.Sequence)
})
```

### Consumer Creation

```go
// Create a consumer with builder pattern
consumerConfig := nats.NewConsumerConfigBuilder("MATCH_STREAM", "match_processor").
    WithSubject("match.found").
    WithDeliverPolicy(jetstream.DeliverAllPolicy).
    WithAckPolicy(jetstream.AckExplicitPolicy).
    WithMaxDeliver(3).
    WithAckWait(30 * time.Second).
    Build()

// Push-based consumer (automatic message delivery)
consumer, err := nats.NewJetStreamConsumer(client, consumerConfig, func(msg jetstream.Msg) error {
    // Process the message
    log.Printf("Received: %s", string(msg.Data()))
    
    // Return error to trigger retry, nil to acknowledge
    return nil
})
defer consumer.Stop()

// Pull-based consumer (manual message fetching)
pullConsumer, err := nats.NewJetStreamPullConsumer(client, consumerConfig)
defer pullConsumer.Stop()

// Process messages in batches
err = pullConsumer.ProcessBatch(10, 5*time.Second, func(msg jetstream.Msg) error {
    // Process each message in the batch
    return nil
})
```

## Service Integration

### Automatic Setup for Services

```go
// Create default consumers for a service
err = nats.CreateDefaultConsumersForService(client, "users")
if err != nil {
    log.Fatal(err)
}

// Start consuming with predefined configurations
matchConsumer, err := nats.NewJetStreamConsumer(client, 
    nats.DefaultConsumerConfigs()["match_found_consumer"],
    handleMatchFoundEvent)
```

### Custom Stream Creation

```go
// Create a custom stream
customStream := nats.NewStreamConfigBuilder("NOTIFICATIONS_STREAM").
    WithSubjects("notification.email", "notification.sms", "notification.push").
    WithRetention(jetstream.WorkQueuePolicy).
    WithStorage(jetstream.FileStorage).
    WithMaxAge(24 * time.Hour).
    WithMaxBytes(100 * 1024 * 1024).
    Build()

err = client.CreateOrUpdateStream(customStream)
```

## Error Handling and Monitoring

### Consumer Monitoring

```go
// Check consumer status
if !consumer.IsActive() {
    log.Warn("Consumer is not active")
}

// Get pending message counts
pending, err := consumer.GetPendingMessages()
ackPending, err := consumer.GetAckPending()

log.Printf("Pending: %d, Ack Pending: %d", pending, ackPending)
```

### Stream Management

```go
// List all streams
streams, err := client.ListStreams()
for _, stream := range streams {
    log.Printf("Stream: %s, Messages: %d, Bytes: %d", 
        stream.Config.Name, stream.State.Msgs, stream.State.Bytes)
}

// Get detailed stream information
streamInfo, err := client.GetStreamInfo("USER_STREAM")
log.Printf("Consumers: %d, First Seq: %d, Last Seq: %d", 
    streamInfo.State.Consumers, streamInfo.State.FirstSeq, streamInfo.State.LastSeq)

// Purge stream (remove all messages)
err = client.PurgeStream("USER_STREAM")

// Delete stream
err = client.DeleteStream("CUSTOM_STREAM")
```

## Backward Compatibility

The new implementation maintains full backward compatibility with existing code:

```go
// These methods work exactly as before
client.Publish("subject", data)
client.Subscribe("subject", handler)
client.Request("subject", data)
client.GetConn() // Access to underlying NATS connection
```

## Configuration

### Environment Variables

```bash
NATS_URL=nats://localhost:4222
NATS_CLUSTER_ID=nebengjek-cluster
NATS_CLIENT_ID=service-name-instance
```

### Stream Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `Retention` | Message retention policy | `LimitsPolicy` |
| `Storage` | Storage type (File/Memory) | `FileStorage` |
| `Replicas` | Number of stream replicas | `1` |
| `MaxAge` | Maximum message age | `24h` |
| `MaxBytes` | Maximum stream size in bytes | `100MB` |
| `MaxMsgs` | Maximum number of messages | `1000000` |
| `Discard` | Discard policy when limits reached | `DiscardOld` |

### Consumer Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `DeliverPolicy` | Message delivery policy | `DeliverAllPolicy` |
| `AckPolicy` | Acknowledgment policy | `AckExplicitPolicy` |
| `AckWait` | Acknowledgment timeout | `30s` |
| `MaxDeliver` | Maximum delivery attempts | `3` |
| `ReplayPolicy` | Message replay policy | `ReplayInstantPolicy` |
| `MaxAckPending` | Max unacknowledged messages | `1000` |

## Best Practices

### 1. Message Design
- Use structured JSON messages with consistent schemas
- Include correlation IDs for request tracing
- Add timestamps for debugging and monitoring

### 2. Consumer Design
- Keep message processing idempotent
- Use appropriate acknowledgment timeouts
- Implement proper error handling and logging

### 3. Stream Design
- Choose appropriate retention policies for your use case
- Set reasonable limits to prevent resource exhaustion
- Use memory storage only for high-frequency, short-lived data

### 4. Error Handling
- Implement circuit breakers for external dependencies
- Use dead letter queues for poison messages
- Monitor consumer lag and processing rates

### 5. Performance
- Use batch processing for high-throughput scenarios
- Configure appropriate consumer concurrency
- Monitor memory usage with memory storage streams

## Troubleshooting

### Common Issues

1. **Consumer not receiving messages**
   - Check stream and consumer configuration
   - Verify subject filters match published subjects
   - Ensure consumer is active and not paused

2. **Messages being redelivered**
   - Check acknowledgment logic in message handlers
   - Verify AckWait timeout is appropriate
   - Look for processing errors causing NAKs

3. **High memory usage**
   - Review memory storage stream configurations
   - Check for consumer lag causing message buildup
   - Monitor MaxAckPending settings

4. **Connection issues**
   - Verify NATS server is running and accessible
   - Check network connectivity and firewall rules
   - Review connection timeout and retry settings

### Monitoring Commands

```go
// Connection status
connected := client.IsConnected()
stats := client.Stats()

// Stream health
streamInfo, _ := client.GetStreamInfo("USER_STREAM")
log.Printf("Stream health: %+v", streamInfo.State)

// Consumer health
consumerInfo, _ := consumer.GetInfo()
log.Printf("Consumer health: %+v", consumerInfo)
```

## Migration Guide

### From Basic NATS to JetStream

1. **Update client creation** (no changes needed - backward compatible)
2. **Add stream configurations** for new features
3. **Migrate critical consumers** to JetStream for reliability
4. **Update monitoring** to include stream and consumer metrics
5. **Test thoroughly** in staging environment

### Gradual Migration Strategy

1. Start with new features using JetStream
2. Migrate critical message flows one by one
3. Keep legacy subscriptions during transition
4. Monitor performance and reliability improvements
5. Complete migration when confident

## Examples

See `examples.go` for comprehensive usage examples including:
- Basic publishing and consuming
- Service integration patterns
- Stream management operations
- Error handling strategies
- Async publishing patterns
- Backward compatibility demonstrations

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review NATS JetStream documentation
3. Monitor logs for error patterns
4. Use the provided monitoring tools