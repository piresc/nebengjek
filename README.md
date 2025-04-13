# NebengJek

A lightweight, real-time trip-hailing and social matching platform integrated with MyTelkomsel.  
**Key Features**: MSISDN-based auth, driver-customer matching, dynamic pricing, and role-based workflows.

---

## 📋 Table of Contents
- [Architecture Overview](#-architecture-overview)
- [Tech Stack](#-tech-stack)
- [Services Breakdown](#-services-breakdown)
- [Data Flow](#-data-flow)
- [Deployment](#-deployment)
- [Testing](#-testing)
- [Configuration](#-configuration)
- [Security](#-security)
- [Scalability](#-scalability)
- [Assumptions](#-assumptions)
- [Contributing](#-contributing)

---

## 🏗️ Architecture Overview

*Microservices-based backend optimized for low latency and scalability.*

### Core Components
| **Component**        | **Description**                                                                 |
|-----------------------|---------------------------------------------------------------------------------|
| **API Gateway**       | Routes requests, validates JWT tokens, and enforces rate limits (Kong/APISIX). |
| **User Service**      | Manages MSISDN-based auth, OTP verification, and role separation (driver/customer). |
| **Location Service**  | Handles real-time geospatial data using PostgreSQL (PostGIS) and Redis caching. |
| **Matching Service**  | Matches drivers/customers within 1 km using NATS for event-driven communication. |
| **Billing Service**   | Calculates fares (3000 IDR/km) and processes payments with Telkomsel’s 5% fee. |
| **Notification Service** | Sends SMS/push alerts via Telkomsel’s APIs.                                  |

---

## 🛠️ Tech Stack
- **Language**: Go (Golang)
- **Databases**: 
  - PostgreSQL (+ PostGIS for geospatial queries)
  - Redis (caching and real-time location indexing)
- **Messaging**: NATS (lightweight message broker)
- **API Framework**: Echo
- **Infrastructure**: 
  - Docker & Kubernetes (deployment)
  - AWS/GCP (cloud hosting)
- **Security**: JWT, HTTPS/TLS

---

## 🔍 Services Breakdown
### 1. **User Service**  
- **Endpoints**:  
  - `POST /auth/login`: Generates OTP via SMS.  
  - `POST /auth/verify`: Validates OTP and issues JWT.  
- **Key Features**:  
  - Role-based access control (`driver`/`customer`).  
  - PostgreSQL schema for user data with role-specific tables.  

### 2. **Location Service**  
- **Endpoints**:  
  - `POST /locations`: Stores driver/customer coordinates.  
  - `GET /locations/nearby`: Finds nearby drivers using PostGIS.  
- **Key Features**:  
  - Real-time Redis caching for 1 km radius queries.  

### 3. **Matching Service**  
- **Workflow**:  
  - Listens to NATS topics (`location_updates`, `match_requests`).  
  - Triggers notifications on successful matches.  

### 4. **Billing Service**  
- **Logic**:  
  - Calculates fare based on distance (PostGIS `ST_Distance`).  
  - Allows drivers to adjust final charges (100% or lower).  

---

## 🌐 Data Flow
1. **Driver activates beacon** → Updates `is_available` in PostgreSQL.  
2. **Customer requests trip** → Matching Service polls nearby drivers via PostGIS.  
3. **Match confirmed** → Notification Service sends SMS.  
4. **Trip ends** → Billing Service computes fare and deducts 5% admin fee.  

---

## 🚀 Deployment
### Docker & Kubernetes

---

## 🧪 Testing
### Run Tests
### Unit tests

### Test Cases
- ✅ OTP validation  
- ✅ Driver-customer matching logic  
- ✅ PostGIS distance calculations  

---

## 🔒 Security
- **JWT Tokens**: Role-based claims for endpoint access.  
- **Rate Limiting**: 10 requests/minute for OTP endpoints.  
- **Encryption**: TLS for APIs; encrypted fields in PostgreSQL.  

---

## 📈 Scalability
- **Auto-Scaling**: Kubernetes HPA for high-traffic services.  
- **Caching**: Redis reduces PostgreSQL load for location queries.  

---

## 📌 Assumptions
- Telkomsel provides SMS/APIs for OTP and notifications.  
- Drivers/customers enable background location sharing.  
- PostgreSQL instance has PostGIS extension enabled.  

---

## 🤝 Contributing
1. Fork the repository.  
2. Create a feature branch (`git checkout -b feature/your-idea`).  
3. Commit changes (`git commit -m 'Add feature'`).  
4. Push to the branch (`git push origin feature/your-idea`).  
5. Open a Pull Request.  

---

📄 **License**: MIT

so the userflow would be, user and driver login using user services(http), and then user and driver turn the beacon, meaning that they want to get find any customer or driver, these using matching-services to matching between these 2(cust and driver), and then one of the driver confirm which user will they get(nats? http?), and then after that confirm in the matching-services, both of them start tracking their gps location using location-services(web-socket), and send nats too? to the trip services? or later?, and then when they are in the destionation, both of them closee the web-socket, and then send the nats to trip-services? and payment-serivices? and after that send it nats to trip services?