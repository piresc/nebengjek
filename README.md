asd
# NebengJek

A lightweight, real-time trip-hailing and social matching platform integrated with MyTelkomsel.  
**Key Features**: MSISDN-based auth, driver-customer matching, dynamic pricing, and role-based workflows.

---

## 📋 Table of Contents
- [Architecture Overview](#-architecture-overview)
- [Tech Stack](#-tech-stack)
- [Services Breakdown](#-services-breakdown)
- [Data Flow](#-data-flow)
- [Database Schema](#-database-schema)
- [Deployment](#-deployment)
- [Testing](#-testing)
- [Security](#-security)
- [Scalability](#-scalability)
- [Assumptions](#-assumptions)
- [Contributing](#-contributing)

---

## 🏗️ System Architecture

### Architecture Overview

The system follows a **4-layer microservices architecture**:

1. **Client Layer**: Mobile apps, web clients, WebSocket connections
2. **Microservices Layer**: 4 core services (Users:9990, Location:9994, Match:9993, Rides:9992)
3. **Message Broker Layer**: NATS for event-driven communication
4. **Data Layer**: PostgreSQL (persistent data) + Redis (caching/real-time)

### System Flow Diagrams

#### 1. Authentication Flow

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

#### 2. Driver Availability & Location Flow

```
┌──────────┐  1. WebSocket Connect   ┌─────────────┐    2. Beacon Event     ┌────────────────┐
│  Mobile  ├────────────────────────▶│ User Service├─────────────────────▶  │ Match Service  │
│  Client  │                         └──────┬──────┘                        └────────────────┘
│          │                                │
│          │ 3. Location Updates            │       4. Location Event      ┌────────────────┐
│          ├────────────────────────────────┼─────────────────────────────▶│Location Service│
└──────────┘                                │                              └────────────────┘
```

#### 3. Ride Matching Flow

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

#### 4. Complete Ride Lifecycle

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

### Clean Architecture Implementation

Each microservice follows **Clean Architecture** principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            Service Architecture                                 │
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                        Handler Layer                                   │    │
│  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────────────┐   │    │
│  │  │  HTTP Handler   │ │ WebSocket       │ │    NATS Handler         │   │    │
│  │  │                 │ │ Handler         │ │                         │   │    │
│  │  │ • REST APIs     │ │ • Real-time     │ │ • Event processing      │   │    │
│  │  │ • Request/      │ │   communication │ │ • Pub/Sub messaging     │   │    │
│  │  │   Response      │ │ • Bidirectional │ │ • Async operations      │   │    │
│  │  │ • Validation    │ │   messaging     │ │                         │   │    │
│  │  └─────────────────┘ └─────────────────┘ └─────────────────────────┘   │    │
│  └─────────────────────────────┬───────────────────────────────────────────┘    │
│                                │                                                │
│                                ▼                                                │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                        Use Case Layer                                  │    │
│  │                                                                         │    │
│  │  • Business Logic Implementation                                        │    │
│  │  • Application-specific rules                                           │    │
│  │  • Orchestrates data flow between entities                              │    │
│  │  • Independent of external concerns                                     │    │
│  │                                                                         │    │
│  └─────────────────────────────┬───────────────────────────────────────────┘    │
│                                │                                                │
│                                ▼                                                │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                    Repository & Gateway Layer                          │    │
│  │  ┌─────────────────────────┐    ┌─────────────────────────────────────┐ │    │
│  │  │     Repository          │    │         Gateway                     │ │    │
│  │  │                         │    │                                     │ │    │
│  │  │ • Data persistence      │    │ • External service communication    │ │    │
│  │  │ • Database operations   │    │ • HTTP client calls                 │ │    │
│  │  │ • Query implementation  │    │ • Third-party integrations          │ │    │
│  │  │ • Data mapping          │    │ • API abstractions                  │ │    │
│  │  └─────────────────────────┘    └─────────────────────────────────────┘ │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Core Components

| **Service**           | **Port** | **Responsibilities**                                                           | **Key Features**                                    |
|-----------------------|----------|--------------------------------------------------------------------------------|----------------------------------------------------|
| **Users Service**     | 9990     | User management, authentication, driver registration, WebSocket connections    | • MSISDN-based OTP auth<br>• JWT token management<br>• Real-time notifications<br>• Driver/passenger role management |
| **Location Service**  | 9994     | Geospatial operations, location tracking, proximity calculations              | • Redis-based location caching<br>• Real-time location updates<br>• Geohash implementation |
| **Match Service**     | 9993     | Driver-passenger matching, match proposals, confirmation handling             | • Event-driven matching algorithm<br>• NATS-based communication<br>• Match state management |
| **Rides Service**     | 9992     | Ride lifecycle management, billing, payment processing                        | • Ride state tracking<br>• Billing calculations<br>• Payment processing with admin fees |

### Technology Stack

| **Category**          | **Technology**                    | **Purpose**                                        |
|-----------------------|-----------------------------------|----------------------------------------------------|  
| **Language**          | Go 1.23                          | Primary development language                       |
| **Web Framework**     | Echo v4                          | HTTP server and routing                            |
| **Database**          | PostgreSQL with pgx driver       | Primary data storage                               |
| **Cache/Session**     | Redis                            | Caching, session storage, geospatial indexing     |
| **Message Broker**    | NATS                             | Event-driven communication between services        |
| **WebSocket**         | Gorilla WebSocket                | Real-time bidirectional communication             |
| **Authentication**    | JWT (golang-jwt/jwt)             | Stateless authentication                           |
| **Database Migration**| Custom SQL migrations            | Database schema management                         |
| **Testing**           | Testify, GoMock                  | Unit and integration testing                       |
| **Containerization** | Docker & Docker Compose          | Service containerization and orchestration        |

## Entity-Relationship Diagram (ERD)

```
┌────────────────────┐          ┌────────────────────┐            ┌────────────────────┐
│      users         │          │      drivers       │            │      locations     │
├────────────────────┤          ├────────────────────┤            ├────────────────────┤
│ id (PK)            │◄─────────┤ user_id (PK, FK)   │            │ id (PK)            │
│ msisdn             │          │ vehicle_type       │            │ user_id (FK)       │
│ fullname           │          │ vehicle_plate      │            │ latitude           │
│ role               │          │                    │            │ longitude          │
│ created_at         │          │                    │            │ updated_at         │
│ updated_at         │          │                    │            │                    │
│ is_active          │          │                    │            │                    │
└────────┬───────────┘          └────────────────────┘            └────────────────────┘
         │                                                                   ▲
         │                                                                   │
         │                                                                   │
         │                                                                   │
         │                        ┌────────────────────┐                     │
         │                        │     matches        │                     │
         │                        ├────────────────────┤                     │
         │                        │ id (PK)            │                     │
         │     ┌──────────────────┤ driver_id (FK)     │                     │
         │     │                  │ passenger_id (FK)  ├─────────────────────┘
         │     │                  │ status             │
         │     │                  │ created_at         │
         │     │                  │ updated_at         │
         │     │                  └────────┬───────────┘
         │     │                           │
         │     │                           │
         │     │                           │
         │     │                           │
┌────────▼─────▼─────┐            ┌────────▼───────────┐            ┌────────────────────┐
│       rides        │            │  billing_ledger    │            │      payments      │
├────────────────────┤            ├────────────────────┤            ├────────────────────┤
│ ride_id (PK)       │            │ entry_id (PK)      │            │ payment_id (PK)    │
│ driver_id (FK)     │◄───────────┤ ride_id (FK)       │◄───────────┤ ride_id (FK)       │
│ customer_id (FK)   │            │ distance           │            │ adjusted_cost      │
│ status             │            │ cost               │            │ admin_fee          │
│ total_cost         │            │ created_at         │            │ driver_payout      │
│ created_at         │            │                    │            │ created_at         │
│ updated_at         │            │                    │            │                    │
└────────────────────┘            └────────────────────┘            └────────────────────┘
```

---

## 🛠️ Architecture Patterns

### Clean Architecture
Each service implements Clean Architecture with distinct layers:
- **Handler Layer**: HTTP, WebSocket, and NATS handlers for external communication
- **Use Case Layer**: Business logic implementation, independent of external concerns
- **Repository Layer**: Data persistence and database operations
- **Gateway Layer**: External service communication and third-party integrations

### Event-Driven Architecture
- **NATS Message Broker**: Enables asynchronous communication between services
- **Pub/Sub Pattern**: Services publish events and subscribe to relevant topics
- **Event Sourcing**: Critical events are captured and processed asynchronously

### Dependency Injection
- **Interface-based Design**: All dependencies are defined as interfaces
- **Mock Generation**: Automated mock generation using GoMock for testing
- **Testable Architecture**: Easy unit testing through dependency injection

---

## 🔍 Service Details

### Users Service (Port: 9990)
**Architecture Layers:**
- **HTTP Handler**: REST API endpoints for user operations
- **WebSocket Handler**: Real-time notifications and location updates
- **NATS Handler**: Event processing for match notifications
- **Use Cases**: User registration, authentication, driver management
- **Repository**: PostgreSQL operations for user and driver data
- **Gateway**: External service integrations

**Key Features:**
- MSISDN-based OTP authentication with Telkomsel validation
- JWT token generation and validation
- Driver registration with vehicle information
- Real-time WebSocket connections for notifications
- Beacon status management for driver availability

### Location Service (Port: 9994)
**Architecture Layers:**
- **NATS Handler**: Location update event processing
- **Use Cases**: Location storage and geospatial operations
- **Repository**: Redis-based location caching and PostgreSQL persistence

**Key Features:**
- Real-time location tracking and updates
- Geohash-based proximity calculations
- Redis geospatial indexing for fast queries
- Location history management

### Match Service (Port: 9993)
**Architecture Layers:**
- **NATS Handler**: Beacon and finder event processing
- **Use Cases**: Driver-passenger matching algorithm
- **Repository**: Match state management in PostgreSQL and Redis
- **Gateway**: Communication with other services

**Key Features:**
- Event-driven matching algorithm
- Driver pool management
- Match proposal and confirmation workflow
- Real-time match notifications via NATS

### Rides Service (Port: 9992)
**Architecture Layers:**
- **HTTP Handler**: Ride management API endpoints
- **NATS Handler**: Match confirmation and ride event processing
- **Use Cases**: Ride lifecycle and billing management
- **Repository**: Ride and payment data persistence
- **Gateway**: External payment service integration

**Key Features:**
- Complete ride lifecycle management
- Dynamic billing calculation based on distance
- Payment processing with 5% admin fee
- Ride history and analytics

---

## 🌐 Data Flow & Communication Patterns

### 1. Authentication Flow
```
Client → Users Service (HTTP) → OTP Generation → JWT Token → Client
```

### 2. Driver Availability Flow
```
Driver App → Users Service (WebSocket) → Beacon Event → NATS → Match Service
```

### 3. Location Update Flow
```
Client → Users Service (WebSocket) → Location Update → NATS → Location Service → Redis
```

### 4. Ride Matching Flow
```
Passenger Request → Match Service → Driver Pool Query → Match Proposal → NATS → 
Users Service → WebSocket → Driver App → Confirmation → NATS → Rides Service
```

### 5. Ride Execution Flow
```
Ride Start → Rides Service → Location Tracking → Billing Calculation → 
Payment Processing → Completion Notification
```

### Inter-Service Communication
- **Synchronous**: HTTP REST APIs for direct service-to-service calls
- **Asynchronous**: NATS messaging for event-driven communication
- **Real-time**: WebSocket connections for client notifications
- **Data Sharing**: Shared PostgreSQL database with service-specific schemas

---

## 💾 Data Architecture

### Database Strategy
- **PostgreSQL**: Primary database for persistent data storage
- **Redis**: Caching layer and real-time data storage
- **Database per Service**: Each service manages its own data schema

### Core Data Models

#### Users Service Schema
```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    msisdn VARCHAR(15) UNIQUE NOT NULL,
    fullname VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'passenger',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    rating DECIMAL(3,2) DEFAULT 0.0
);

-- Drivers table
CREATE TABLE drivers (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    vehicle_type VARCHAR(50) NOT NULL,
    vehicle_plate VARCHAR(20) NOT NULL
);
```

#### Location Service Schema
```sql
-- Locations stored in Redis with geospatial indexing
-- Key pattern: location:{user_id}
-- Value: {latitude, longitude, timestamp}
```

#### Match Service Schema
```sql
-- Matches table
CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL,
    passenger_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Rides Service Schema
```sql
-- Rides table
CREATE TABLE rides (
    ride_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    total_cost DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Billing ledger
CREATE TABLE billing_ledger (
    entry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID REFERENCES rides(ride_id),
    distance DECIMAL(10,2) NOT NULL,
    cost DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Payments
CREATE TABLE payments (
    payment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID REFERENCES rides(ride_id),
    adjusted_cost DECIMAL(10,2) NOT NULL,
    admin_fee DECIMAL(10,2) NOT NULL,
    driver_payout DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 🚀 Deployment Architecture

### Containerization Strategy
```yaml
# Docker Compose Services
services:
  users-service:    # Port 9990
  location-service: # Port 9994  
  match-service:    # Port 9993
  rides-service:    # Port 9992
  postgres:         # Port 5432
  redis:           # Port 6379
  nats:            # Port 4222
```

### Service Dependencies
- **All Services** depend on: PostgreSQL, Redis, NATS
- **Health Checks**: Each service implements `/ping` endpoint
- **Graceful Startup**: Services wait for dependencies to be healthy
- **Auto-restart**: Services automatically restart on failure

### Configuration Management
- **Environment Files**: Service-specific `.env` files
- **Docker Secrets**: Sensitive data management
- **Volume Mounts**: Log persistence and configuration sharing

### Networking
- **Internal Network**: Services communicate via Docker network
- **Port Mapping**: External access to specific service ports
- **Service Discovery**: DNS-based service resolution

---

## 🧪 Testing Strategy

### Testing Architecture
- **Unit Tests**: Business logic testing with mocked dependencies
- **Integration Tests**: End-to-end service testing with real dependencies
- **Mock Generation**: Automated mock creation using GoMock

### Test Coverage Areas
```
├── Handler Layer Tests
│   ├── HTTP endpoint testing
│   ├── WebSocket connection testing
│   └── NATS message handling testing
├── Use Case Layer Tests
│   ├── Business logic validation
│   ├── Error handling scenarios
│   └── Edge case coverage
├── Repository Layer Tests
│   ├── Database operation testing
│   ├── Redis operation testing
│   └── Data consistency validation
└── Integration Tests
    ├── Service-to-service communication
    ├── End-to-end workflow testing
    └── Performance testing
```

### Testing Tools
- **Testify**: Assertion library and test suites
- **GoMock**: Mock generation and dependency injection
- **Miniredis**: In-memory Redis for testing
- **SQL Mock**: Database operation mocking
- **NATS Test Server**: Message broker testing

---

## 🔒 Security Implementation

### Authentication & Authorization
- **JWT Tokens**: Stateless authentication with configurable expiration
- **MSISDN Validation**: Telkomsel number format validation
- **OTP Verification**: Secure one-time password authentication
- **Role-based Access**: Driver vs passenger permission separation

### API Security
- **Input Validation**: Request payload validation and sanitization
- **Rate Limiting**: Protection against API abuse and DDoS
- **CORS Configuration**: Cross-origin request security
- **Request Logging**: Comprehensive audit trail

### Data Security
- **Database Encryption**: Encrypted connections to PostgreSQL
- **Redis Security**: Secured cache access with authentication
- **Environment Variables**: Sensitive configuration management
- **Connection Pooling**: Secure database connection management

### Communication Security
- **TLS/HTTPS**: Encrypted client-server communication
- **WebSocket Security**: Secure real-time connections
- **NATS Security**: Authenticated message broker communication
- **Internal Network**: Isolated service communication

---

## 📈 Scalability & Performance

### Horizontal Scaling
- **Stateless Services**: Each service can be scaled independently
- **Load Balancing**: Multiple instances behind load balancers
- **Database Connection Pooling**: Efficient database resource utilization
- **Redis Clustering**: Distributed caching for high availability

### Performance Optimization
- **Geospatial Indexing**: Redis-based location queries for sub-millisecond response
- **Connection Reuse**: HTTP client connection pooling
- **Async Processing**: NATS-based event-driven architecture
- **Caching Strategy**: Multi-layer caching (Redis + in-memory)

### Monitoring & Observability
- **Health Checks**: Comprehensive service health monitoring
- **Structured Logging**: JSON-based logging with correlation IDs
- **Metrics Collection**: Service performance and business metrics
- **Distributed Tracing**: Request flow tracking across services

### Resource Management
- **Memory Optimization**: Efficient Go garbage collection
- **CPU Utilization**: Goroutine-based concurrent processing
- **Database Optimization**: Query optimization and indexing
- **Network Efficiency**: Minimal payload sizes and compression

---

## 📌 Technical Assumptions & Constraints

### External Dependencies
- **Telkomsel Integration**: SMS/API services for OTP delivery and notifications
- **Location Services**: GPS-enabled devices with background location sharing
- **Network Connectivity**: Stable internet connection for real-time features

### Infrastructure Requirements
- **PostgreSQL**: Database with UUID extension support
- **Redis**: Version 6+ with geospatial command support
- **NATS**: Message broker for event-driven communication
- **Docker**: Container runtime for service deployment

### Business Rules
- **Fare Calculation**: Base rate of 3000 IDR per kilometer
- **Admin Fee**: Fixed 5% commission on all rides
- **MSISDN Format**: Telkomsel numbers only (prefix validation)
- **Driver Availability**: Real-time beacon status management

### Performance Assumptions
- **Concurrent Users**: Designed for thousands of concurrent connections
- **Response Time**: Sub-second response for critical operations
- **Data Retention**: Configurable retention policies for historical data
- **Geographic Scope**: Optimized for Indonesian geographic coordinates

---

## 🚀 Deployment

### Production Deployment Pipeline

The project uses GitHub Actions for automated deployment triggered by releases:

#### Release Deployment Workflow

1. **Automated Testing**: Unit tests run with Redis service container
2. **SonarCloud Analysis**: Code quality and security scanning
3. **Multi-Service Build**: All four services built as Go binaries
4. **Docker Image Creation**: Each service containerized with version tags
5. **EC2 Deployment**: Automated deployment to production EC2 instance

#### Docker Image Strategy

```yaml
# Each service gets tagged with:
- latest                    # Latest stable version
- v1.0.0                   # Specific release version
```

#### Production Infrastructure

- **Platform**: AWS EC2
- **Orchestration**: Docker Compose
- **Registry**: Docker Hub
- **Deployment**: SSH-based automated deployment

### Environment Configuration

Required secrets for deployment:

```yaml
DOCKER_USERNAME          # Docker Hub username
DOCKER_PASSWORD          # Docker Hub password
SSH_PRIVATE_KEY          # EC2 SSH private key
EC2_HOST                 # Production server host
EC2_USERNAME             # EC2 server username
SONA_TOKEN              # SonarCloud authentication
```

## 📊 CI/CD & Code Quality

### Continuous Integration Pipeline

#### Pull Request Workflow

1. **Automated Testing**
   - Unit tests with race condition detection
   - Redis integration testing
   - Coverage report generation

2. **Code Quality Analysis**
   - SonarCloud static analysis
   - Security vulnerability scanning
   - Code smell detection

3. **Coverage Comparison**
   - PR branch vs master branch coverage
   - Automated PR comments with coverage diff
   - Detailed coverage reports

#### SonarCloud Configuration

```properties
# Quality Gate Settings
sonar.projectKey=piresc_nebengjek
sonar.organization=nebengjek-prod
sonar.host.url=https://sonarcloud.io

# Coverage Analysis
sonar.go.coverage.reportPaths=coverage.txt
sonar.coverage.exclusions=**/cmd/**/main.go,**/mocks/**

# Code Exclusions
sonar.exclusions=**/*_test.go,**/vendor/**,**/bin/**
```

### Code Quality Metrics

- **Test Coverage**: Tracked and compared across branches
- **Code Smells**: Automated detection via SonarCloud
- **Security Hotspots**: Vulnerability scanning
- **Maintainability**: Technical debt analysis

### Branch Protection

- **Required Status Checks**: All tests must pass
- **Coverage Requirements**: Maintained or improved coverage
- **Code Review**: Required before merge
- **SonarCloud Quality Gate**: Must pass quality checks

## 🤝 Development Workflow

### Getting Started
1. **Clone Repository**: `git clone <repository-url>`
2. **Install Dependencies**: `go mod download`
3. **Setup Environment**: Copy and configure `.env` files for each service
4. **Start Infrastructure**: `docker-compose up postgres redis nats`
5. **Run Migrations**: Execute SQL migration files in `db/migrations/`
6. **Start Services**: Run individual services or use `docker-compose up`

### Development Guidelines
1. **Clean Architecture**: Follow established layer separation
2. **Interface Design**: Define interfaces before implementations
3. **Test Coverage**: Write tests for all business logic
4. **Mock Generation**: Use `go generate` for mock creation
5. **Code Review**: All changes require peer review

### Project Structure
```
├── cmd/                    # Service entry points
├── internal/pkg/           # Shared internal packages
├── services/               # Service implementations
│   ├── {service}/handler/  # Handler layer
│   ├── {service}/usecase/  # Business logic layer
│   ├── {service}/repository/ # Data access layer
│   └── {service}/gateway/  # External service layer
├── db/migrations/          # Database migrations
├── docs/                   # Documentation
├── .github/workflows/     # CI/CD pipelines
└── docker-compose.yml      # Container orchestration
```

### Development Best Practices

- **Testing**: Write unit tests for all business logic
- **Coverage**: Maintain or improve test coverage
- **Code Quality**: Follow SonarCloud recommendations
- **Documentation**: Update docs for new features
- **Commits**: Use conventional commit messages

### Contributing
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-idea`)
3. Follow coding standards and write tests
4. Commit changes (`git commit -m 'Add feature'`)
5. Push to the branch (`git push origin feature/your-idea`)
6. Open a Pull Request

---

📄 **License**: MIT
