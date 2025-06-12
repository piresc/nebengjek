# NebengJek: Lite, Simple, and Fast Ojek Online Platform

## 1. Business Problem & Solution
**Core Value**: Real-time driver-rider matching with 1km radius, 3000 IDR/km pricing, 5% admin fee
**Key Flows**: Driver beacon → Customer request → Smart matching → Trip execution → Automated billing
**Differentiators**: WebSocket real-time communication, Redis geospatial optimization, event-driven architecture

## 2. Architecture Overview
**Services**: Users, Location, Match, Rides (4 microservices)
**Tech Stack**: Go 1.23+, Echo, PostgreSQL, Redis, NATS JetStream, Docker
**Communication**: WebSocket (7 event types) + NATS messaging + HTTP APIs
**Observability**: New Relic APM, structured logging, health monitoring



## 3. Key Implementation Features

**Matching Algorithm**:
1. Passenger WebSocket: finder_update → NATS event → Match Service
2. Redis GEORADIUS query (5km radius, max 5 drivers)
3. Distance sorting + availability filtering → Match proposal
4. 30-second timeout for driver acceptance → Ride creation

**Real-Time Communication**: 7 WebSocket events (beacon, finder, match, location, ride, payment, connection)
**Location Tracking**: 1-minute GPS updates, Redis geospatial indexes, session correlation
**Billing**: GPS-based calculation, driver adjustment (≤100%), 5% admin fee, real-time ledger

## 4. Security & Quality

**Security**: API key authentication, JWT WebSocket auth, GitLeaks scanning, SonarCloud analysis
**Testing**: GoMock unit tests, SQL mocks, WebSocket testing, NATS event simulation
**CI/CD**: GitHub Actions, secret scanning, automated deployment, health validation
**Monitoring**: New Relic APM, structured logging, performance metrics (<100ms WebSocket)



## 5. Architecture Decisions & Trade-offs

**Technology Choices**:
- **NATS JetStream** vs Kafka: Lightweight operations, built-in persistence
- **Redis + PostgreSQL** vs MongoDB: ACID compliance, geospatial support
- **Echo WebSocket** vs SSE: Bidirectional communication, native integration
- **Unified WebSocket** vs Separate Service: Operational simplicity, planned extraction

**Key Assumptions**:
- Location updates: 1-minute intervals (configurable)
- Search radius: 5km default, 30-second driver timeout
- Pricing: 3000 IDR/km base, driver adjustment ≤100%
- Data retention: 24h NATS events, 7-day ride history

## 6. Roadmap & Next Steps

**Immediate (Q2 2025)**:
- Integration and load testing with realistic traffic simulation
- Extract WebSocket service from Users Service for independent scaling
- Kubernetes deployment with auto-scaling and orchestration
- Enhanced configuration with Viper and Google Secret Manager

**Advanced Features**:
- New Relic custom dashboards and quantitative management
- ML-based matching algorithms for preference learning
- Multi-region deployment for geographic scaling
- Enhanced observability with distributed tracing