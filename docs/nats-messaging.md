# NATS Messaging System

## Overview

NebengJek uses NATS JetStream for asynchronous event-driven communication between microservices. All streams use `WorkQueuePolicy` (messages deleted after ACK) with `FileStorage` for persistence.

## Streams

| Stream | Subjects | Purpose |
|--------|----------|---------|
| `LOCATION` | `location.>` | Real-time location updates and tracking |
| `MATCH` | `match.>` | Driver-passenger matching events |
| `RIDE` | `ride.>` | Ride lifecycle management |
| `USER` | `user.>` | User management and beacon events |
| `NOTIFICATION` | `notification.>` | Gateway-bound user notifications |

All streams: 24h max age, file storage, work queue retention.

## Service Communication Map

```
Users/Gateway ──publish──→ location.update, match.request, user.beacon
                ←subscribe── match.found, match.confirmed, ride.*, payment.*, notification.deliver

Location      ──publish──→ location.stored
                ←subscribe── location.update

Match         ──publish──→ match.found, match.confirmed, match.cancelled
                ←subscribe── match.request, match.response, user.beacon, location.update

Rides         ──publish──→ ride.created, ride.started, ride.completed, payment.requested
                ←subscribe── match.confirmed

Notification  ──publish──→ notification.deliver
                ←subscribe── match.*, ride.*, payment.* (selective)
```

## Event Schemas

### Location Events

**`location.update`** — User sends GPS position
```json
{
  "user_id": "uuid",
  "user_type": "driver|passenger",
  "location": { "latitude": -6.2088, "longitude": 106.8456, "accuracy": 10.5 },
  "ride_id": "uuid",
  "timestamp": "2025-01-08T10:00:00Z"
}
```

**`location.driver.available`** — Driver availability change
```json
{
  "driver_id": "uuid",
  "is_available": true,
  "location": { "latitude": -6.2088, "longitude": 106.8456, "geohash": "qqgux4" },
  "vehicle_info": { "type": "motorcycle", "license_plate": "B1234XYZ" }
}
```

### Match Events

**`match.request`** — Passenger requests a ride
```json
{
  "match_request_id": "uuid",
  "passenger_id": "uuid",
  "pickup_location": { "latitude": -6.2088, "longitude": 106.8456 },
  "destination_location": { "latitude": -6.2200, "longitude": 106.8300 },
  "preferences": { "vehicle_type": "motorcycle", "max_distance_km": 5.0 }
}
```

**`match.found`** — Match service found a driver
```json
{
  "match_id": "uuid",
  "driver_id": "uuid",
  "passenger_id": "uuid",
  "estimates": { "distance_km": 3.2, "duration_minutes": 15, "fare_estimate": 9600 },
  "expires_at": "2025-01-08T10:05:00Z"
}
```

**`match.response`** — Driver/passenger accepts or rejects
```json
{
  "match_id": "uuid",
  "user_id": "uuid",
  "user_type": "driver|passenger",
  "response": "accepted|rejected",
  "reason": "too_far|price_too_high|other"
}
```

**`match.confirmed`** — Both parties accepted
```json
{
  "match_id": "uuid",
  "driver_id": "uuid",
  "passenger_id": "uuid",
  "pickup_location": { "latitude": -6.2088, "longitude": 106.8456 },
  "destination_location": { "latitude": -6.2200, "longitude": 106.8300 }
}
```

### Ride Events

**`ride.created`** — New ride from confirmed match
```json
{
  "ride_id": "uuid",
  "match_id": "uuid",
  "driver_id": "uuid",
  "passenger_id": "uuid",
  "pricing": { "base_rate_per_km": 3000, "estimated_fare": 9600 }
}
```

**`ride.completed`** — Ride finished with billing
```json
{
  "ride_id": "uuid",
  "driver_id": "uuid",
  "passenger_id": "uuid",
  "trip_summary": { "total_distance_km": 3.2, "total_duration_minutes": 15 },
  "billing": {
    "base_fare": 9600,
    "adjustment_factor": 0.9,
    "adjusted_fare": 8640,
    "admin_fee_percent": 5.0,
    "admin_fee": 432,
    "final_fare": 8208
  }
}
```

### Payment Events

**`payment.requested`** — Payment needed for completed ride
```json
{
  "payment_id": "uuid",
  "ride_id": "uuid",
  "passenger_id": "uuid",
  "amount": 8208,
  "payment_methods": ["wallet", "bank_transfer"]
}
```

**`payment.completed`** / **`payment.failed`** — Payment result
```json
{
  "payment_id": "uuid",
  "ride_id": "uuid",
  "amount": 8208,
  "payment_method": "wallet",
  "transaction_id": "uuid"
}
```

### Notification Events

**`notification.deliver`** — Notification service → Gateway for WebSocket delivery
```json
{
  "user_id": "uuid",
  "type": "match_proposal|ride_started|ride_completed|payment_processed",
  "data": { ... },
  "timestamp": "2025-01-08T10:00:00Z"
}
```

## Consumer Configuration

All consumers use explicit ACK, max 3 delivery attempts, durable names:

```go
// Example: Match service consuming beacon events
js.Subscribe("user.beacon.*", handler, nats.Durable("match-beacon-consumer"))
```

### Consumer Naming Convention
`{service}-{event-type}-consumer`, e.g. `users-match-consumer`, `match-location-consumer`

## Event Flow: Complete Ride

```
Gateway  →  match.request  →  Match Service
Match    →  match.found    →  Gateway (WebSocket to driver)
Gateway  →  match.response →  Match Service
Match    →  match.confirmed → Gateway + Rides Service
Rides    →  ride.created   →  Gateway
Rides    →  ride.started   →  Gateway
Rides    →  ride.completed →  Gateway + Notification
Rides    →  payment.requested → Gateway
Payment  →  payment.completed → Gateway + Rides
```

## Error Handling

- **Explicit ACK/NAK** on all consumers
- **Max 3 retries** with exponential backoff before dead letter
- **Idempotent processing** — consumers handle duplicate delivery
- Messages include `event_id` for deduplication

## Configuration

```bash
NATS_URL=nats://localhost:4222
JETSTREAM_ENABLED=true
```

Streams are auto-created on service startup via `internal/pkg/nats/jetstream_helpers.go`.
