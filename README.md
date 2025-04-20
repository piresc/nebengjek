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

## 🏗️ Architecture Overview

*Microservices-based backend optimized for low latency and scalability.*

### High-Level Design Architecture

```
┌───────────────────────────────────────────────────────────────────────────┐
│                                                                           │
│                          Load Balancer / API Gateway                      │
│                                                                           │
└───────────┬───────────────────┬───────────────────┬────────────────────┬──┘
            │                   │                   │                    │
            ▼                   ▼                   ▼                    ▼
┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐ ┌────────────────────┐
│                   │ │                   │ │                   │ │                    │
│   User Service    │ │ Location Service  │ │   Match Service   │ │   Rides Service    │
│                   │ │                   │ │                   │ │                    │
└────────┬──────────┘ └───────┬───────────┘ └────────┬──────────┘ └─────────┬──────────┘
         │                    │                      │                      │
         │                    │                      │                      │
┌────────┴────────────────────┴──────────────────────┴──────────────────────┴──────────┐
│                                                                                      │
│                                  Message Broker (NATS)                               │
│                                                                                      │
└──────────┬───────────────────────────────────┬──────────────────────────────────┬────┘
           │                                   │                                  │
           ▼                                   ▼                                  ▼
┌────────────────────┐              ┌────────────────────┐             ┌────────────────────┐
│                    │              │                    │             │                    │
│   Redis Cluster    │              │   PostgreSQL DB    │             │ Logging/Monitoring │
│                    │              │                    │             │                    │
└────────────────────┘              └────────────────────┘             └────────────────────┘
```

### Low-Level Design Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                          Client Applications                          │
│                                                                      │
│               ┌───────────────────┐   ┌───────────────────┐          │
│               │  Mobile App       │   │  Web Client       │          │
│               └─────────┬─────────┘   └─────────┬─────────┘          │
└───────────────────────┬─┴───────────────────────┴──────────────────┬─┘
                        │                                            │
                        │  HTTP/WebSocket                            │
                        ▼                                            ▼
┌────────────────────────────────────────────────────────────────────────┐
│                                                                        │
│                         API Gateway / Load Balancer                    │
│                                                                        │
│  ┌────────────────────┐  ┌────────────────────┐  ┌──────────────────┐  │
│  │  JWT Validation    │  │  Rate Limiting     │  │  Request Routing │  │
│  └────────────────────┘  └────────────────────┘  └──────────────────┘  │
└────────────────────────────────┬───────────────────────────────────────┘
                                 │
                                 │
         ┌─────────────────────────────────────────────────┐
         │                                                 │
         ▼                                                 ▼
┌─────────────────┐                              ┌─────────────────────────┐
│  User Service   │                              │                         │
├─────────────────┤                              │      NATS Messaging     │
│                 │                              │                         │
│ ┌─────────────┐ │                              │  ┌──────────────────┐  │
│ │HTTP Handlers│ │                              │  │Message Producers │  │
│ └─────────────┘ │                              │  └──────────────────┘  │
│                 │                              │                         │
│ ┌─────────────┐ │                              │  ┌──────────────────┐  │
│ │WebSocket    │ │                              │  │Message Consumers │  │
│ │Handlers     │ │                              │  └──────────────────┘  │
│ └─────────────┘ │                              │                         │
│                 │                              │  ┌──────────────────┐  │
│ ┌─────────────┐ │◄─────┐                       │  │Message Subjects  │  │
│ │NATS Handlers│ │      │                       │  └──────────────────┘  │
│ └─────────────┘ │      │                       └─────────────────────────┘
└─────────────────┘      │                                  ▲
                         │                                  │
                         │                                  │
┌─────────────────┐      │                                  │
│Location Service │      │                                  │
├─────────────────┤      │                                  │
│                 │      │                                  │
│ ┌─────────────┐ │      │                                  │
│ │Geo Spatial  │ │      │                                  │
│ │Query        │ │      │                                  │
│ └─────────────┘ │      │                                  │
│                 │      │                                  │
│ ┌─────────────┐ │      │                                  │
│ │Redis Geo    │ │      │                                  │
│ │Index        │ │      │                                  │
│ └─────────────┘ │      │                                  │
│                 │      │                                  │
│ ┌─────────────┐ │◄─────┼──────────────────────────────────┘
│ │NATS Handlers│ │      │                                  │
│ └─────────────┘ │      │                                  │
└─────────────────┘      │                                  │
                         │                                  │
                         │                                  │
┌─────────────────┐      │                                  │
│ Match Service   │      │                                  │
├─────────────────┤      │                                  │
│                 │      │                                  │
│ ┌─────────────┐ │      │                                  │
│ │Driver       │ │      │                                  │
│ │Matching Algo│ │      │                                  │
│ └─────────────┘ │      │                                  │
│                 │      │                                  │
│ ┌─────────────┐ │      │                                  │
│ │Match        │ │      │                                  │
│ │Repository   │ │      │                                  │
│ └─────────────┘ │      │                                  │
│                 │      │                                  │
│ ┌─────────────┐ │◄─────┼──────────────────────────────────┘
│ │NATS Handlers│ │      │
│ └─────────────┘ │      │
└─────────────────┘      │
                         │
                         │
┌─────────────────┐      │
│ Rides Service   │      │
├─────────────────┤      │
│                 │      │
│ ┌─────────────┐ │      │
│ │Ride         │ │      │
│ │Lifecycle    │ │      │
│ └─────────────┘ │      │
│                 │      │
│ ┌─────────────┐ │      │
│ │Billing      │ │      │
│ │Calculator   │ │      │
│ └─────────────┘ │      │
│                 │      │
│ ┌─────────────┐ │◄─────┘
│ │NATS Handlers│ │
│ └─────────────┘ │
└─────────────────┘

┌─────────────────────────┐  ┌───────────────────────┐  ┌─────────────────────┐
│                         │  │                       │  │                     │
│  PostgreSQL Database    │  │  Redis Cluster        │  │  Monitoring &       │
│  • User data            │  │  • Caching            │  │  Logging            │
│  • Driver data          │  │  • Geo index          │  │  • Centralized logs │
│  • Ride history         │  │  • Real-time location │  │  • Service metrics  │
│  • Billing ledger       │  │  • Session storage    │  │  • Health checks    │
│                         │  │                       │  │                     │
└─────────────────────────┘  └───────────────────────┘  └─────────────────────┘
```

### Core Components
| **Component**        | **Description**                                                                 |
|-----------------------|---------------------------------------------------------------------------------|
| **User Service**      | Manages MSISDN-based auth, OTP verification, and role separation (driver/customer). |
| **Location Service**  | Handles real-time geospatial data using PostgreSQL (PostGIS) and Redis caching. |
| **Match Service**     | Matches drivers/customers within proximity using NATS for event-driven communication. |
| **Ride Service**      | Manages ride lifecycle from creation to completion. |
| **API Gateway**       | Routes requests, validates JWT tokens, and enforces rate limits. |

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

## 🛠️ Tech Stack
- **Language**: Go (Golang)
- **Frameworks & Libraries**:
  - Echo (HTTP framework)
  - sqlx (Database operations)
  - NATS (Message broker)
- **Databases**: 
  - PostgreSQL (+ PostGIS for geospatial queries)
  - Redis (caching and real-time location indexing)
- **Messaging**: NATS (lightweight pub/sub message broker)
- **Infrastructure**: 
  - Docker & Docker Compose (containerization)
- **Security**: JWT, API Key middleware

---

## 🔍 Services Breakdown
### 1. **User Service**  
- **Features**:  
  - OTP-based authentication
  - User profile management
  - Driver registration and management
  - Beacon status management (driver availability)
- **Key Components**:
  - Authentication handlers
  - User repository
  - Driver repository
  - OTP generation and verification

### 2. **Location Service**  
- **Features**:  
  - Location tracking
  - Nearby driver discovery
- **Key Components**:
  - Geospatial queries
  - Real-time location updates
  - Redis-based location caching

### 3. **Match Service**  
- **Features**:  
  - Driver-passenger matching
  - Match proposal and confirmation
  - Real-time matching notifications
- **Key Components**:
  - Match repository
  - NATS event handling
  - Redis-based match storage

### 4. **Ride Service**  
- **Features**:  
  - Ride lifecycle management
  - Billing and payment processing
  - Fare calculation
- **Key Components**:
  - Ride creation and tracking
  - Billing ledger
  - Payment processing with admin fee calculation

---

## 🌐 Data Flow
1. **User Authentication**
   - User requests OTP via phone number
   - System verifies OTP and issues JWT token
   
2. **Driver Availability**
   - Driver toggles beacon status
   - Location service updates driver availability
   
3. **Ride Request & Matching**
   - Passenger requests ride with location data
   - Match service finds nearby available drivers
   - System sends match proposals to drivers
   - Driver accepts the match

4. **Ride Execution & Payment**
   - Ride service creates ride entry
   - System tracks ride progress and distance
   - On completion, billing service calculates fare
   - Payment is processed with 5% admin fee

---

## 💾 Database Schema

### Core Entities
- **Users**: Basic user information and authentication
  - Fields: `user_id`, `phone_number`, `name`, `role`, `created_at`
  
- **Drivers**: Driver-specific information
  - Fields: `driver_id`, `user_id`, `license_number`, `vehicle_info`
  
- **Rides**: Ride tracking information
  - Fields: `ride_id`, `driver_id`, `customer_id`, `status`, `total_cost`, `created_at`, `updated_at`
  
- **Billing Ledger**: Individual billing entries
  - Fields: `entry_id`, `ride_id`, `distance`, `cost`, `created_at`
  
- **Payments**: Payment records
  - Fields: `payment_id`, `ride_id`, `adjusted_cost`, `admin_fee`, `driver_payout`, `created_at`
  
- **Locations**: Real-time location data
  - Fields: `location_id`, `user_id`, `latitude`, `longitude`, `updated_at`

---

## 🚀 Deployment
### Docker & Docker Compose
- Containerized services for easy deployment
- Multi-container setup with PostgreSQL and Redis
- Service discovery via container networking
- Environment-specific configuration via .env files

---

## 🧪 Testing
- Unit tests for critical business logic
- Integration tests for API endpoints
- Test cases covering:
  - User authentication
  - Driver-customer matching
  - Ride completion and billing

---

## 🔒 Security
- **JWT Authentication**: Secure API access via tokens
- **API Key Middleware**: Service-to-service authentication
- **Rate Limiting**: Protect against abuse
- **Input Validation**: Prevent injection attacks

---

## 📈 Scalability
- **Microservices Architecture**: Independent scaling of services
- **Event-Driven Design**: Asynchronous processing via NATS
- **Caching Strategy**: Redis for frequent queries
- **Stateless Services**: Horizontal scaling capability

---

## 📌 Assumptions
- Telkomsel provides SMS/APIs for OTP and notifications  
- Drivers/customers enable background location sharing
- PostgreSQL instance has PostGIS extension enabled
- System calculates fare at a rate of 3000 IDR/km
- Admin fee is fixed at 5% of the fare

---

## 🤝 Contributing
1. Fork the repository  
2. Create a feature branch (`git checkout -b feature/your-idea`)  
3. Commit changes (`git commit -m 'Add feature'`)  
4. Push to the branch (`git push origin feature/your-idea`)  
5. Open a Pull Request  

---

📄 **License**: MIT
