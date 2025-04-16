# Nebengjek Microservices Docker Setup

This document provides instructions for running the Nebengjek microservices using Docker Compose.

## Services Overview

The Nebengjek application consists of the following services:

1. **User Service** - Handles user authentication and profile management
2. **Billing Service** - Manages payment processing and invoicing
3. **Location Service** - Tracks and manages location data
4. **Match Service** - Matches triprs with drivers
5. **ride Service** - Manages ride requests and status

## Infrastructure Services

- **PostgreSQL** - Primary database for all services
- **Redis** - Caching and session management
- **NATS** - Message broker for inter-service communication

## Prerequisites

- Docker and Docker Compose installed on your system
- Git repository cloned to your local machine

## Getting Started

### Running the Services

1. Navigate to the project root directory:
   ```
   cd nebengjek
   ```

2. Start all services using Docker Compose:
   ```
   docker-compose up
   ```

   To run in detached mode:
   ```
   docker-compose up -d
   ```

3. To start specific services only:
   ```
   docker-compose up <service-name>
   ```
   Example: `docker-compose up user-service billing-service`

### Stopping the Services

```
docker-compose down
```

To remove volumes as well:
```
docker-compose down -v
```

## Service Ports

| Service | Port |
|---------|--------|
| User Service | 9995 |
| Billing Service | 9996 |
| Location Service | 9998 |
| Match Service | 10000 |
| rides Service | 10002 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| NATS | 4222, 8222 |

## Environment Configuration

Each service has its own environment configuration file located in its respective directory under `cmd/<service-name>/.env.example`. For production use, create a copy of these files without the `.example` suffix and update the values accordingly.

## Logs

Service logs are mounted to the `./logs` directory in the project root.

## Troubleshooting

- **Service fails to start**: Check the logs for the specific service using `docker-compose logs <service-name>`
- **Database connection issues**: Ensure PostgreSQL is running and healthy with `docker-compose ps postgres`
- **Network issues**: Verify all services are on the same network with `docker network inspect nebengjek_local`

## Development Workflow

For development purposes, you can rebuild and restart a specific service after code changes:

```
docker-compose up -d --build <service-name>
```

Example: `docker-compose up -d --build user-service`