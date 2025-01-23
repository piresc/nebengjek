# NebengJek Backend

A microservices-based backend system for NebengJek, a ride-sharing feature within the MyTelkomsel app.

## Architecture Overview

### High-Level Design

The system consists of four main microservices:
1. Auth Service - Handles user authentication and profile management
2. Match Service - Manages driver-customer matching within 1km radius
3. Trip/Billing Service - Handles trip management and cost calculation
4. Payment Service - Processes payments and settlements

### Technology Stack

- Node.js with Express.js for microservices
- PostgreSQL for data storage
- Redis for caching and real-time location data
- Docker for containerization
- Nginx for API Gateway/Load Balancer

## Project Structure

```
nebengjek-backend/
├─ auth-service/
├─ match-service/
├─ trip-billing-service/
├─ payment-service/
├─ infrastructure/
└─ docs/
```

## Getting Started

### Prerequisites

- Node.js (v16 or later)
- Docker and Docker Compose
- PostgreSQL
- Redis

### Setup Instructions

1. Clone the repository
```bash
git clone https://github.com/YourUsername/nebengjek-backend.git
cd nebengjek-backend
```

2. Install dependencies for each service
```bash
cd auth-service && npm install
cd ../match-service && npm install
cd ../trip-billing-service && npm install
cd ../payment-service && npm install
```

3. Set up environment variables
- Copy `.env.example` to `.env` in each service directory
- Update the variables with your configuration

4. Start the services using Docker Compose
```bash
docker-compose up
```

## API Documentation

Detailed API documentation for each service can be found in the `docs` directory.

## Testing

Each service includes its own test suite. To run tests:

```bash
npm test
```

## License

MIT