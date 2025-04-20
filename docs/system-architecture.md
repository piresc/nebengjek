# NebengJek System Architecture

## Overview

NebengJek is a microservices-based trip-hailing platform. The system is composed of four core services that communicate through both synchronous (HTTP) and asynchronous (NATS) channels. This document provides a comprehensive overview of the system architecture, communication patterns, and data flows.

## Service Architecture

![NebengJek Architecture](https://via.placeholder.com/800x600?text=NebengJek+Architecture+Diagram)

### Core Services

| Service | Primary Responsibility | Communication Protocols |
|---------|------------------------|------------------------|
| **User Service** | User management, authentication, real-time client communication | HTTP, WebSocket, NATS |
| **Location Service** | Geospatial tracking and queries | NATS |
| **Match Service** | Driver-passenger pairing algorithms | NATS |
| **Rides Service** | Ride lifecycle, billing, payments | NATS |

## Communication Patterns

NebengJek employs different communication patterns based on the specific requirements of each interaction:

### Synchronous Communication (HTTP)

Used for:
- User authentication
- Profile management
- Driver registration
- Direct requests requiring immediate response

### Real-time Bidirectional Communication (WebSocket)

Used for:
- Real-time location updates
- Match notifications
- Ride status updates
- Driver beacon status changes

### Asynchronous Communication (NATS)

Used for:
- Service-to-service communication
- Event broadcasting
- Location tracking
- Ride updates and billing events

## Event Flow Diagrams

### 1. Authentication Flow

```
┌──────────┐     1. Login Request     ┌─────────────┐
│  Mobile  ├────────────────────────▶ │ User Service│
│  Client  │                          └──────┬──────┘
│          │ 2. OTP Verification               │
│          │◀───────────────────────────────── │
│          │                                   │
│          │ 3. JWT Token                      │
└──────────┘◀──────────────────────────────────┘
```

### 2. Beacon Activation & Location Flow

```
┌──────────┐  1. WebSocket Connect   ┌─────────────┐    2. Beacon Event     ┌────────────────┐
│  Mobile  ├────────────────────────▶│ User Service├─────────────────────▶  │ Match Service  │
│  Client  │                         └──────┬──────┘                        └────────────────┘
│          │                                │
│          │ 3. Location Updates            │       4. Location Event      ┌────────────────┐
│          ├────────────────────────────────┼─────────────────────────────▶│Location Service│
└──────────┘                                │                              └────────────────┘
```

### 3. Match Flow

```
┌──────────┐  1. Match Request       ┌─────────────┐    2. Match Request     ┌────────────────┐
│ Passenger├────────────────────────▶│ User Service├─────────────────────────▶│ Match Service  │
└──────────┘                         └─────────────┘                          │                │
                                                                              │ 3. Find nearby │
┌──────────┐                         ┌─────────────┐  5. Match Proposal       │    drivers     │
│  Driver  │◀───────────────────────▶│ User Service│◀────────────────────────┤                │
└──────────┘                         └─────────────┘                          │ 4. Location    │
     │                                      ▲                                 │    query       │
     │                                      │                                 │                │
     │ 6. Accept Match                      │ 7. Match Accepted              └───────┬────────┘
     └──────────────────────────────────────┴─────────────────────────────────────────┘
```

### 4. Ride Flow

```
┌──────────┐                         ┌─────────────┐     Match Accepted     ┌────────────────┐
│  Driver  │                         │ User Service│                        │ Match Service  │
└──────────┘                         └─────────────┘                        └───────┬────────┘
     │                                      │                                       │
     │ Location Updates                     │                                       │
     ├──────────────────────────────────────┼───────────────────────────┐           │
     │                                      │                           │           │
     │                                      │                           ▼           │
     │                                      │                   ┌───────────────┐   │
     │                                      │                   │Location Service│   │
     │                                      │                   └───────┬───────┘   │
     │                                      │                           │           │
     │                                      │                           │           │
     │                                      │      Ride Created         ▼           │
┌────┴─────┐                         ┌──────┴────┐◀──────────────┌──────────────┐◀──┘
│ Passenger│◀────────────────────────│User Service│               │ Ride Service │
└──────────┘  Ride Update Events     └─────────────┘              └──────┬───────┘
                                                                         │
                                                                         │
                   ┌───────────────────────────────────────────────────┐ │
                   │                                                   │ │
                   │  Fare calculation, billing ledger updates,        │◀┘
                   │  trip completion, payment processing              │
                   │                                                   │
                   └───────────────────────────────────────────────────┘
```

## Data Storage

### PostgreSQL

The primary relational database for persistent storage:

| Service | Tables | Purpose |
|---------|--------|---------|
| User Service | `users`, `drivers`, `otp_codes` | User profiles, authentication |
| Match Service | `matches` | Match proposals and history |
| Rides Service | `rides`, `billing_ledger`, `payments` | Ride tracking and billing |
| Location Service | `locations` | Historical location data |

### Redis

Used for caching, real-time data, and spatial indexing:

| Service | Data Structures | Purpose |
|---------|-----------------|---------|
| User Service | Key-value store | OTP storage, session cache |
| Match Service | Lists, Sorted sets | Active drivers, match proposals |
| Location Service | Geo-indexes | Real-time location tracking |

## API Gateway and Security

### Authentication

- MSISDN-based authentication with OTP verification
- JWT tokens for authenticated API access
- API keys for service-to-service communication

### Security Measures

- Rate limiting on authentication endpoints
- Input validation and sanitization
- JWT token verification middleware
- HTTPS encryption for all API endpoints

## Scalability Considerations

The NebengJek architecture is designed to scale horizontally:

- Stateless services allow for multiple instances
- NATS supports message load balancing across service instances
- Redis clustering for distributed caching
- Database read replicas for query scaling

## Monitoring and Observability

The system includes:

- Structured logging across all services
- Performance metrics for critical operations
- Request tracing for cross-service operations
- Health check endpoints for service monitoring

## Deployment

NebengJek services are containerized with Docker and can be deployed:

- As individual containers for development
- With Docker Compose for integrated testing
- On Kubernetes for production environments

## Failure Handling

The system incorporates several patterns for resilience:

- Circuit breakers for external service calls
- Retry mechanisms with exponential backoff
- Dead letter queues for failed events
- Graceful degradation of non-critical features