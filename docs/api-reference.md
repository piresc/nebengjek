# NebengJek API Reference

## Overview

This document provides comprehensive API documentation for all NebengJek microservices. Each service exposes REST APIs for synchronous operations and WebSocket endpoints for real-time communication.

## Authentication

### JWT Authentication
Most endpoints require JWT authentication obtained through the OTP verification process.

**Header Format**:
```
Authorization: Bearer <jwt_token>
```

### API Key Authentication
Service-to-service communication uses API key authentication.

**Header Format**:
```
X-API-Key: <service_api_key>
```

## Global Response Formats

### Success Response
```json
{
  "status": "success",
  "data": { ... },
  "timestamp": "2025-01-08T10:00:00Z",
  "request_id": "uuid"
}
```

### Error Response
```json
{
  "status": "error",
  "error": "Error message",
  "code": "ERROR_CODE",
  "timestamp": "2025-01-08T10:00:00Z",
  "request_id": "uuid"
}
```

## Users Service API (Port: 9990)

### Health Endpoints

#### GET /health
Basic health check for load balancers.

**Response**:
```json
{
  "status": "ok",
  "service": "users-service",
  "timestamp": "2025-01-08T10:00:00Z"
}
```

#### GET /health/detailed
Comprehensive health check with dependency status.

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-08T10:00:00Z",
  "service": "users-service",
  "version": "1.0.0",
  "dependencies": {
    "postgres": {
      "status": "healthy"
    },
    "redis": {
      "status": "healthy"
    },
    "nats": {
      "status": "healthy"
    }
  }
}
```

#### GET /health/ready
Kubernetes readiness probe endpoint.

**Response (Healthy)**:
```json
{
  "status": "ready",
  "service": "users-service"
}
```

**Response (Unhealthy)**: `503 Service Unavailable`

#### GET /health/live
Kubernetes liveness probe endpoint.

**Response**:
```json
{
  "status": "alive",
  "service": "users-service"
}
```

### Authentication Endpoints

#### POST /auth/otp/generate
Generate OTP for phone number authentication.

**Request**:
```json
{
  "msisdn": "+628123456789"
}
```

**Response**:
```json
{
  "status": "success",
  "message": "OTP sent successfully",
  "expires_in": 300
}
```

**Error Responses**:
- `400 Bad Request`: Invalid MSISDN format
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: SMS service unavailable

#### POST /auth/otp/verify
Verify OTP and obtain JWT token.

**Request**:
```json
{
  "msisdn": "+628123456789",
  "otp": "123456"
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "token": "jwt token.",
    "expires_in": 86400,
    "user": {
      "id": "uuid",
      "msisdn": "+628123456789",
      "role": "passenger",
      "created_at": "2025-01-08T10:00:00Z"
    }
  }
}
```

**Error Responses**:
- `400 Bad Request`: Invalid OTP or expired
- `404 Not Found`: MSISDN not found
- `500 Internal Server Error`: Authentication service error

### User Management Endpoints

#### POST /users
Create a new user (requires JWT).

**Headers**:
```
Authorization: Bearer <jwt_token>
```

**Request**:
```json
{
  "msisdn": "+628123456789",
  "name": "John Doe",
  "email": "john@example.com"
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "id": "uuid",
    "msisdn": "+628123456789",
    "name": "John Doe",
    "email": "john@example.com",
    "role": "passenger",
    "created_at": "2025-01-08T10:00:00Z",
    "updated_at": "2025-01-08T10:00:00Z"
  }
}
```

#### GET /users/:id
Retrieve user by ID (requires JWT).

**Headers**:
```
Authorization: Bearer <jwt_token>
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "id": "uuid",
    "msisdn": "+628123456789",
    "name": "John Doe",
    "email": "john@example.com",
    "role": "passenger",
    "created_at": "2025-01-08T10:00:00Z",
    "updated_at": "2025-01-08T10:00:00Z"
  }
}
```

**Error Responses**:
- `404 Not Found`: User not found
- `403 Forbidden`: Access denied

### Driver Management Endpoints

#### POST /drivers/register
Register user as driver (requires JWT).

**Headers**:
```
Authorization: Bearer <jwt_token>
```

**Request**:
```json
{
  "vehicle_type": "motorcycle",
  "vehicle_brand": "Honda",
  "vehicle_model": "Vario 150",
  "vehicle_year": 2023,
  "license_plate": "B1234XYZ",
  "driver_license": "1234567890123456"
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "id": "uuid",
    "user_id": "uuid",
    "vehicle_type": "motorcycle",
    "vehicle_brand": "Honda",
    "vehicle_model": "Vario 150",
    "vehicle_year": 2023,
    "license_plate": "B1234XYZ",
    "driver_license": "1234567890123456",
    "status": "active",
    "created_at": "2025-01-08T10:00:00Z"
  }
}
```

### WebSocket Endpoint

#### GET /ws
Real-time bidirectional communication (requires JWT).

**Headers**:
```
Authorization: Bearer <jwt_token>
Upgrade: websocket
Connection: Upgrade
```

**Connection Flow**:
1. Client connects with JWT token in Authorization header
2. Server validates JWT and establishes WebSocket connection
3. Client can send/receive real-time events

**Supported Events**: See [WebSocket Events Specification](websocket-events-specification.md)

## Location Service API (Port: 9994)

### Health Endpoints
Same as Users Service health endpoints.

### Location Management Endpoints

#### POST /locations/update
Update user location (requires API key).

**Headers**:
```
X-API-Key: <location_service_api_key>
```

**Request**:
```json
{
  "user_id": "uuid",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "accuracy": 10.5,
  "timestamp": "2025-01-08T10:00:00Z"
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "location_id": "uuid",
    "user_id": "uuid",
    "latitude": -6.2088,
    "longitude": 106.8456,
    "accuracy": 10.5,
    "geohash": "qqgux4",
    "timestamp": "2025-01-08T10:00:00Z"
  }
}
```

#### GET /locations/nearby
Find nearby users within radius (requires API key).

**Headers**:
```
X-API-Key: <location_service_api_key>
```

**Query Parameters**:
- `latitude` (required): Center latitude
- `longitude` (required): Center longitude
- `radius_km` (optional): Search radius in kilometers (default: 5.0)
- `user_type` (optional): Filter by user type (driver/passenger)
- `limit` (optional): Maximum results (default: 50)

**Example**: `/locations/nearby?latitude=-6.2088&longitude=106.8456&radius_km=2.0&user_type=driver&limit=10`

**Response**:
```json
{
  "status": "success",
  "data": {
    "center": {
      "latitude": -6.2088,
      "longitude": 106.8456
    },
    "radius_km": 2.0,
    "total_found": 3,
    "users": [
      {
        "user_id": "uuid",
        "latitude": -6.2100,
        "longitude": 106.8450,
        "distance_km": 0.15,
        "user_type": "driver",
        "last_updated": "2025-01-08T10:00:00Z"
      }
    ]
  }
}
```

## Match Service API (Port: 9993)

### Health Endpoints
Same as Users Service health endpoints.

### Match Management Endpoints

#### POST /matches/request
Request driver-passenger match (requires API key).

**Headers**:
```
X-API-Key: <match_service_api_key>
```

**Request**:
```json
{
  "passenger_id": "uuid",
  "pickup_location": {
    "latitude": -6.2088,
    "longitude": 106.8456,
    "address": "Jl. Sudirman No. 1, Jakarta"
  },
  "destination_location": {
    "latitude": -6.2200,
    "longitude": 106.8300,
    "address": "Jl. Thamrin No. 10, Jakarta"
  },
  "preferences": {
    "max_distance_km": 5.0,
    "vehicle_type": "motorcycle"
  }
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "match_request_id": "uuid",
    "passenger_id": "uuid",
    "pickup_location": {
      "latitude": -6.2088,
      "longitude": 106.8456,
      "address": "Jl. Sudirman No. 1, Jakarta"
    },
    "destination_location": {
      "latitude": -6.2200,
      "longitude": 106.8300,
      "address": "Jl. Thamrin No. 10, Jakarta"
    },
    "status": "searching",
    "created_at": "2025-01-08T10:00:00Z"
  }
}
```

#### POST /matches/:match_id/accept
Accept a match proposal (requires API key).

**Headers**:
```
X-API-Key: <match_service_api_key>
```

**Request**:
```json
{
  "user_id": "uuid",
  "user_type": "driver"
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "match_id": "uuid",
    "status": "confirmed",
    "driver_id": "uuid",
    "passenger_id": "uuid",
    "estimated_pickup_time": 5,
    "estimated_duration": 15,
    "estimated_distance_km": 3.2,
    "confirmed_at": "2025-01-08T10:00:00Z"
  }
}
```

#### GET /matches/:match_id
Get match details (requires API key).

**Headers**:
```
X-API-Key: <match_service_api_key>
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "match_id": "uuid",
    "driver_id": "uuid",
    "passenger_id": "uuid",
    "pickup_location": {
      "latitude": -6.2088,
      "longitude": 106.8456,
      "address": "Jl. Sudirman No. 1, Jakarta"
    },
    "destination_location": {
      "latitude": -6.2200,
      "longitude": 106.8300,
      "address": "Jl. Thamrin No. 10, Jakarta"
    },
    "status": "confirmed",
    "created_at": "2025-01-08T10:00:00Z",
    "confirmed_at": "2025-01-08T10:00:00Z"
  }
}
```

## Rides Service API (Port: 9992)

### Health Endpoints
Same as Users Service health endpoints.

### Ride Management Endpoints

#### POST /rides
Create a new ride (requires API key).

**Headers**:
```
X-API-Key: <rides_service_api_key>
```

**Request**:
```json
{
  "match_id": "uuid",
  "driver_id": "uuid",
  "passenger_id": "uuid",
  "pickup_location": {
    "latitude": -6.2088,
    "longitude": 106.8456,
    "address": "Jl. Sudirman No. 1, Jakarta"
  },
  "destination_location": {
    "latitude": -6.2200,
    "longitude": 106.8300,
    "address": "Jl. Thamrin No. 10, Jakarta"
  }
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "ride_id": "uuid",
    "match_id": "uuid",
    "driver_id": "uuid",
    "passenger_id": "uuid",
    "pickup_location": {
      "latitude": -6.2088,
      "longitude": 106.8456,
      "address": "Jl. Sudirman No. 1, Jakarta"
    },
    "destination_location": {
      "latitude": -6.2200,
      "longitude": 106.8300,
      "address": "Jl. Thamrin No. 10, Jakarta"
    },
    "status": "created",
    "base_rate_per_km": 3000,
    "created_at": "2025-01-08T10:00:00Z"
  }
}
```

#### PUT /rides/:ride_id/start
Start a ride (requires API key).

**Headers**:
```
X-API-Key: <rides_service_api_key>
```

**Request**:
```json
{
  "driver_id": "uuid",
  "start_location": {
    "latitude": -6.2088,
    "longitude": 106.8456
  }
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "ride_id": "uuid",
    "status": "in_progress",
    "start_location": {
      "latitude": -6.2088,
      "longitude": 106.8456
    },
    "started_at": "2025-01-08T10:00:00Z"
  }
}
```

#### PUT /rides/:ride_id/complete
Complete a ride (requires API key).

**Headers**:
```
X-API-Key: <rides_service_api_key>
```

**Request**:
```json
{
  "driver_id": "uuid",
  "end_location": {
    "latitude": -6.2200,
    "longitude": 106.8300
  },
  "adjustment_factor": 0.9
}
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "ride_id": "uuid",
    "status": "completed",
    "end_location": {
      "latitude": -6.2200,
      "longitude": 106.8300
    },
    "total_distance_km": 3.2,
    "total_duration_minutes": 15,
    "base_fare": 9600,
    "adjustment_factor": 0.9,
    "adjusted_fare": 8640,
    "admin_fee": 432,
    "final_fare": 8208,
    "completed_at": "2025-01-08T10:15:00Z"
  }
}
```

#### GET /rides/:ride_id
Get ride details (requires API key).

**Headers**:
```
X-API-Key: <rides_service_api_key>
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "ride_id": "uuid",
    "match_id": "uuid",
    "driver_id": "uuid",
    "passenger_id": "uuid",
    "pickup_location": {
      "latitude": -6.2088,
      "longitude": 106.8456,
      "address": "Jl. Sudirman No. 1, Jakarta"
    },
    "destination_location": {
      "latitude": -6.2200,
      "longitude": 106.8300,
      "address": "Jl. Thamrin No. 10, Jakarta"
    },
    "status": "completed",
    "total_distance_km": 3.2,
    "total_duration_minutes": 15,
    "final_fare": 8208,
    "created_at": "2025-01-08T10:00:00Z",
    "started_at": "2025-01-08T10:00:00Z",
    "completed_at": "2025-01-08T10:15:00Z"
  }
}
```

#### GET /rides/user/:user_id
Get user's ride history (requires API key).

**Headers**:
```
X-API-Key: <rides_service_api_key>
```

**Query Parameters**:
- `limit` (optional): Maximum results (default: 20)
- `offset` (optional): Pagination offset (default: 0)
- `status` (optional): Filter by ride status

**Response**:
```json
{
  "status": "success",
  "data": {
    "user_id": "uuid",
    "total_rides": 25,
    "rides": [
      {
        "ride_id": "uuid",
        "status": "completed",
        "pickup_address": "Jl. Sudirman No. 1, Jakarta",
        "destination_address": "Jl. Thamrin No. 10, Jakarta",
        "final_fare": 8208,
        "completed_at": "2025-01-08T10:15:00Z"
      }
    ],
    "pagination": {
      "limit": 20,
      "offset": 0,
      "has_more": true
    }
  }
}
```

## Error Codes

### Common Error Codes

| Code | Description | HTTP Status |
|------|-------------|-------------|
| `INVALID_REQUEST` | Request validation failed | 400 |
| `UNAUTHORIZED` | Authentication required | 401 |
| `FORBIDDEN` | Access denied | 403 |
| `NOT_FOUND` | Resource not found | 404 |
| `RATE_LIMITED` | Rate limit exceeded | 429 |
| `INTERNAL_ERROR` | Internal server error | 500 |
| `SERVICE_UNAVAILABLE` | Service temporarily unavailable | 503 |

### Service-Specific Error Codes

#### Users Service
| Code | Description |
|------|-------------|
| `INVALID_MSISDN` | Invalid phone number format |
| `OTP_EXPIRED` | OTP has expired |
| `OTP_INVALID` | Invalid OTP code |
| `USER_EXISTS` | User already exists |
| `DRIVER_EXISTS` | User already registered as driver |

#### Location Service
| Code | Description |
|------|-------------|
| `INVALID_COORDINATES` | Invalid latitude/longitude |
| `LOCATION_NOT_FOUND` | Location data not found |
| `RADIUS_TOO_LARGE` | Search radius exceeds maximum |

#### Match Service
| Code | Description |
|------|-------------|
| `NO_DRIVERS_AVAILABLE` | No drivers found in area |
| `MATCH_EXPIRED` | Match request has expired |
| `MATCH_ALREADY_ACCEPTED` | Match already accepted by another user |

#### Rides Service
| Code | Description |
|------|-------------|
| `RIDE_NOT_FOUND` | Ride not found |
| `INVALID_RIDE_STATUS` | Invalid ride status transition |
| `PAYMENT_FAILED` | Payment processing failed |

## Rate Limiting

### Authentication Endpoints
- OTP Generation: 5 requests per minute per MSISDN
- OTP Verification: 10 requests per minute per MSISDN

### API Endpoints
- Standard endpoints: 100 requests per minute per API key
- Location updates: 1000 requests per minute per API key
- Health checks: No rate limiting

## Pagination

List endpoints support pagination using `limit` and `offset` parameters:

**Request**:
```
GET /rides/user/uuid?limit=20&offset=40
```

**Response**:
```json
{
  "data": [...],
  "pagination": {
    "limit": 20,
    "offset": 40,
    "total": 150,
    "has_more": true
  }
}
```

## Related Documentation

- [WebSocket Events Specification](websocket-events-specification.md) - Real-time event documentation
- [NATS Event Schemas](nats-event-schemas.md) - Asynchronous event documentation
- [Security Implementation](security-implementation.md) - Authentication and security details
- [System Architecture](system-architecture.md) - Overall system design