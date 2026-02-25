# WebSocket Guide

## Overview

NebengJek uses WebSocket for real-time bidirectional communication between clients and the Gateway Service. The implementation uses `github.com/coder/websocket` with Redis-backed session storage and NATS broadcasting for multi-gateway horizontal scaling.

## Architecture

```
Client → Gateway (WebSocket) → NATS JetStream → Microservices
                ↕                      ↕
         Redis Sessions         Other Gateways
```

- **Library**: `github.com/coder/websocket` with automatic compression
- **Session storage**: Redis — distributed across gateway instances
- **Broadcasting**: NATS JetStream — cross-gateway user/role targeting
- **Heartbeat**: 30s ping interval, 90s timeout (3 missed pings), auto-cleanup
- **Auth**: JWT token via `Authorization: Bearer <token>` header

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| WebSocket Handler | `services/gateway/handler/websocket/echo_handler.go` | Connection lifecycle, message routing |
| Session Repository | `services/gateway/repository/websocket_session.go` | Redis session CRUD, TTL refresh |
| NATS Broadcaster | `services/gateway/handler/nats/notification.go` | Cross-gateway message delivery |

### Multi-Gateway Support

Each gateway instance gets a unique server ID (`gateway-{hostname}-{timestamp}`). Sessions are stored in Redis with the server ID, enabling:
- Multiple gateway instances behind a load balancer
- NATS broadcasting routes messages to the correct gateway
- Redis TTL auto-expires stale sessions

## Connection Flow

```
1. Client connects with JWT → Gateway validates token
2. Gateway registers session in Redis
3. Gateway starts heartbeat goroutine (ping every 30s)
4. Client sends events → Gateway routes to NATS → Microservices process
5. Microservices publish responses → NATS → Gateway → Client
6. On disconnect: session removed from Redis, client cleaned up
```

## Event Reference

### Connection Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `connection.established` | Server → Client | Connection successful, includes user_id, role |
| `connection.error` | Server → Client | Auth failure or connection error |

### Beacon Events (Driver Availability)

| Event | Direction | Description |
|-------|-----------|-------------|
| `beacon.update` | Client → Server | Toggle driver/passenger availability with location |
| `beacon.status` | Server → Client | Confirmation of beacon status |
| `beacon.nearby_drivers` | Server → Client | List of nearby available drivers for passenger |

**beacon.update payload:**
```json
{
  "type": "beacon.update",
  "payload": {
    "is_active": true,
    "user_type": "driver",
    "location": { "latitude": -6.2088, "longitude": 106.8456 }
  }
}
```

### Location Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `location.update` | Client → Server | GPS update during ride |
| `location.tracking` | Server → Client | Other party's location during ride |

### Match Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `match.request` | Client → Server | Passenger requests a ride |
| `match.proposal` | Server → Client | Match proposal sent to driver + passenger |
| `match.accept` | Client → Server | Accept match proposal |
| `match.reject` | Client → Server | Reject match proposal |
| `match.confirmed` | Server → Client | Both parties accepted |
| `match.cancelled` | Server → Client | Match cancelled (timeout/user/system) |

**match.proposal payload:**
```json
{
  "type": "match.proposal",
  "payload": {
    "match_id": "uuid",
    "driver_id": "uuid",
    "passenger_id": "uuid",
    "estimated_distance_km": 3.2,
    "estimated_fare": 9600,
    "driver_info": {
      "name": "Driver Name",
      "vehicle_type": "motorcycle",
      "license_plate": "B1234XYZ",
      "rating": 4.8
    },
    "expires_at": "2025-01-08T10:05:00Z"
  }
}
```

### Ride Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `ride.started` | Server → Client | Ride has begun |
| `ride.progress` | Server → Client | Distance, duration, current fare |
| `ride.arrived` | Client → Server | Driver signals arrival |
| `ride.completed` | Server → Client | Final billing breakdown |
| `ride.cancelled` | Server → Client | Ride cancelled |

**ride.completed payload:**
```json
{
  "type": "ride.completed",
  "payload": {
    "ride_id": "uuid",
    "total_distance_km": 3.2,
    "total_duration_minutes": 15,
    "billing": {
      "base_fare": 9600,
      "adjustment_factor": 0.9,
      "adjusted_fare": 8640,
      "admin_fee_percent": 5.0,
      "admin_fee": 432,
      "final_fare": 8208
    }
  }
}
```

### Payment Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `payment.request` | Server → Client | Request payment from passenger |
| `payment.method_selected` | Client → Server | Passenger selects payment method |
| `payment.completed` | Server → Client | Payment successful |
| `payment.failed` | Server → Client | Payment failed, retry allowed |

### Error Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `error.validation` | Server → Client | Invalid event payload |
| `error.rate_limit` | Server → Client | Rate limit exceeded |

## Complete Ride Flow

```
Passenger → match.request
                    Server → match.proposal → Driver
Driver → match.accept
                    Server → match.confirmed → Both
                    Server → ride.started → Both
        loop:
            Driver → location.update
                    Server → location.tracking → Passenger
                    Server → ride.progress → Both
Driver → ride.arrived
                    Server → ride.completed → Both
                    Server → payment.request → Passenger
Passenger → payment.method_selected
                    Server → payment.completed → Both
```

## Client Implementation Notes

- Implement automatic reconnection with exponential backoff
- Heartbeat is handled automatically by the library (no client-side ping needed)
- Queue events when offline, sync on reconnect
- Validate outgoing events before sending
- Implement timeouts for events expecting responses
