# NATS JetStream Architecture & Implementation

## Why JetStream?

### The Problem with Basic NATS
Our ride-sharing platform initially used basic NATS messaging, which had several limitations:

1. **No Message Persistence**: Messages were lost if consumers were offline
2. **No Delivery Guarantees**: Fire-and-forget messaging with no acknowledgments
3. **No Message Replay**: Couldn't replay missed messages for new consumers
4. **Limited Scalability**: No built-in load balancing or consumer groups
5. **No Ordering Guarantees**: Messages could arrive out of order

### JetStream Advantages

#### 1. **Message Persistence & Durability**
- Messages are stored on disk/memory with configurable retention policies
- Survives service restarts and network failures
- Critical for ride-sharing where losing a match or ride event is unacceptable

#### 2. **Delivery Guarantees**
- **At-least-once delivery**: Messages are guaranteed to be delivered
- **Exactly-once semantics**: With proper deduplication using message IDs
- **Acknowledgment-based**: Consumers must ACK messages for completion

#### 3. **Message Replay & Recovery**
- New consumers can replay historical messages
- Failed message processing can be retried automatically
- Essential for audit trails and debugging

#### 4. **Horizontal Scalability**
- Multiple consumers can process messages in parallel
- Built-in load balancing across consumer instances
- Supports both work queue and interest-based patterns

#### 5. **Stream Policies & Configuration**
- **Interest Policy**: Multiple consumers get copies (pub/sub pattern)
- **Work Queue Policy**: Messages distributed among consumers (load balancing)
- **Limits Policy**: Retention based on size, age, or message count

## Our JetStream Implementation

### Stream Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   USER_STREAM   │    │  MATCH_STREAM   │    │   RIDE_STREAM   │    │ LOCATION_STREAM │
│                 │    │                 │    │                 │    │                 │
│ user.beacon     │    │ match.found     │    │ ride.pickup     │    │ location.update │
│ user.finder     │    │ match.rejected  │    │ ride.started    │    │ location.aggregate
│                 │    │ match.accepted  │    │ ride.arrived    │    │                 │
│                 │    │                 │    │ ride.completed  │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Dual Consumption Pattern

Our architecture requires **dual consumption** for several events:

#### Match Accepted Event
```
match.accepted → ┌─ match_accepted_users (WebSocket notifications)
                 └─ match_accepted_rides (Create ride in database)
```

#### Ride Pickup Event
```
ride.pickup → ┌─ ride_pickup_users (Notify driver/passenger)
              └─ ride_pickup_match (Lock users from matching)
```

### Consumer Naming Convention

We use the pattern: `{subject}_{service}` to ensure unique consumers:

- `user_beacon_users` / `user_beacon_match`
- `match_accepted_users` / `match_accepted_rides`
- `ride_pickup_users` / `ride_pickup_match`

### Stream Configurations

#### USER_STREAM
- **Retention**: `InterestPolicy` (multiple consumers get copies)
- **Storage**: `FileStorage` (persistent)
- **MaxAge**: 24 hours
- **Subjects**: `user.beacon`, `user.finder`

#### MATCH_STREAM
- **Retention**: `InterestPolicy` (enables dual consumption)
- **Storage**: `FileStorage` (persistent)
- **MaxAge**: 1 hour
- **Subjects**: `match.found`, `match.rejected`, `match.accepted`

#### RIDE_STREAM
- **Retention**: `LimitsPolicy` (audit trail)
- **Storage**: `FileStorage` (persistent)
- **MaxAge**: 7 days
- **Subjects**: `ride.pickup`, `ride.started`, `ride.arrived`, `ride.completed`

#### LOCATION_STREAM
- **Retention**: `InterestPolicy` (real-time processing)
- **Storage**: `MemoryStorage` (fast access)
- **MaxAge**: 2 hours
- **Subjects**: `location.update`, `location.aggregate`

### Message Flow Example

```
1. Driver accepts match
   ↓
2. Users service publishes to match.accepted
   ↓
3. JetStream delivers to both consumers:
   ├─ match_accepted_users → WebSocket notification
   └─ match_accepted_rides → Create ride in DB
   ↓
4. Rides service publishes to ride.pickup
   ↓
5. JetStream delivers to both consumers:
   ├─ ride_pickup_users → Notify users
   └─ ride_pickup_match → Lock users
```

### Error Handling & Reliability

#### Automatic Retries
- Failed messages are automatically retried (MaxDeliver: 3-5)
- Exponential backoff prevents system overload
- Dead letter queues for permanently failed messages

#### Message Deduplication
- Each message has a unique ID: `{event-type}-{entity-id}-{timestamp}`
- Prevents duplicate processing during retries
- 5-minute deduplication window

#### Acknowledgment Patterns
```go
// Success - message is ACKed
if err := handler(msg); err != nil {
    return err // NAK - message will be retried
}
return nil // ACK - message processing complete
```

## Implementation Benefits

### 1. **Fault Tolerance**
- Services can restart without losing messages
- Network partitions don't cause data loss
- Automatic recovery and replay capabilities

### 2. **Scalability**
- Multiple instances of each service can run
- Load balancing across consumer instances
- Horizontal scaling without coordination

### 3. **Observability**
- Built-in metrics for message rates, acknowledgments
- Stream and consumer health monitoring
- Message tracing and audit capabilities

### 4. **Consistency**
- Guaranteed message delivery ensures data consistency
- Ordered processing within subjects
- Transactional semantics with acknowledgments

### 5. **Performance**
- Asynchronous processing reduces latency
- Batching and buffering optimizations
- Memory storage for high-frequency data

## Monitoring & Operations

### Key Metrics
- **Message Rate**: Messages/second per stream
- **Consumer Lag**: Unprocessed messages per consumer
- **Acknowledgment Rate**: Success/failure ratios
- **Stream Size**: Storage usage and retention

### Health Checks
- Stream connectivity and availability
- Consumer processing rates
- Message delivery latencies
- Error rates and retry patterns

## Best Practices Implemented

1. **Idempotent Consumers**: All message handlers are idempotent
2. **Circuit Breakers**: Prevent cascade failures
3. **Graceful Degradation**: Fallback mechanisms for critical paths
4. **Resource Limits**: Bounded queues and memory usage
5. **Security**: TLS encryption and authentication
6. **Testing**: Comprehensive unit and integration tests

This JetStream implementation provides a robust, scalable, and reliable messaging foundation for our ride-sharing platform, ensuring no critical events are lost and enabling seamless horizontal scaling.