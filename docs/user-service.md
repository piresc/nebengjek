# User Service Documentation

## Overview

The User Service is the core service for user management and authentication in the NebengJek system. It handles user registration, authentication via OTP, driver registration, and manages real-time communication with clients through WebSockets.

## Handlers

The User Service implements the following handler types:
1. HTTP Handlers - For REST API endpoints
2. WebSocket Handlers - For real-time bidirectional communication
3. NATS Handlers - For asynchronous event processing

### HTTP Handlers

#### Auth Handler (`services/user/handler/http/auth.go`)

| Endpoint | Method | Description | Request | Response |
|----------|--------|-------------|---------|----------|
| `/auth/otp/generate` | POST | Generates and sends OTP to a phone number | `{"msisdn": "+628123456789"}` | Success message or error |
| `/auth/otp/verify` | POST | Verifies OTP and returns JWT token | `{"msisdn": "+628123456789", "otp": "123456"}` | JWT token or error |

**Implementation Details:**
- `GenerateOTP`: Validates the MSISDN format and calls the user usecase to generate and store an OTP. In a production environment, this would send the OTP via SMS.
- `VerifyOTP`: Validates the OTP for the given MSISDN. If valid, generates a JWT token for the user. If the user doesn't exist, creates a new one with the role "passenger".

#### User Handler (`services/user/handler/http/user.go`)

| Endpoint | Method | Description | Request | Response |
|----------|--------|-------------|---------|----------|
| `/users` | POST | Creates a new user | User object | Created user or error |
| `/users/:id` | GET | Retrieves user by ID | - | User details or error |
| `/drivers/register` | POST | Registers a user as driver | Driver info | Success or error |

**Implementation Details:**
- `CreateUser`: Processes user registration by binding request data to a user model and calling the user usecase.
- `GetUser`: Retrieves user details by ID.
- `RegisterDriver`: Allows existing users to register as drivers by providing additional driver information.

### WebSocket Handler (`services/user/handler/websocket/`)

| Event Type | Description | Payload |
|------------|-------------|---------|
| `beacon.update` | Updates driver/passenger availability | `{"is_active": true, "user_type": "driver"}` |
| `match.accept` | Driver accepts a match proposal | Match proposal object |
| `location.update` | Updates user's real-time location | `{"location": {"latitude": 0.0, "longitude": 0.0}, "ride_id": "uuid"}` |
| `ride.arrived` | Driver signals arrival at destination | `{"ride_id": "uuid", "adjustment_factor": 1.0}` |

**Components:**
1. `WebSocketManager` (`manager.go`): Manages WebSocket connections, authentication, and client state.
   - Key methods: 
     - `HandleWebSocket`: Entry point for new WebSocket connections
     - `authenticateClient`: Authenticates clients using JWT
     - `NotifyClient`: Sends events to specific clients

2. `WebSocketHandlers` (`handlers.go`): Processes various WebSocket message types:
   - `handleBeaconUpdate`: Processes beacon status updates
   - `handleMatchAccept`: Processes match acceptance from drivers
   - `handleLocationUpdate`: Processes location updates from clients
   - `handleRideArrived`: Processes ride arrival events

### NATS Handlers (`services/user/handler/nats/`)

| Subject | Handler | Purpose |
|---------|---------|---------|
| `match.found` | `handleMatchEvent` | Notifies both driver and passenger about a match |
| `match.confirm` | `handleMatchConfirmEvent` | Notifies about match confirmation |
| `match.rejected` | `handleMatchRejectedEvent` | Notifies the driver when a match is rejected |
| `ride.started` | `handleRideStartedEvent` | Notifies driver and passenger when a ride starts |
| `ride.completed` | `handleRideCompletedEvent` | Notifies about ride completion and payment details |

**Implementation Files:**
- `handler.go`: Sets up NATS connections and consumers
- `match.go`: Handles match-related events
- `ride.go`: Handles ride-related events

## User Flow

1. **Authentication**:
   - User requests OTP via `POST /auth/otp/generate`
   - User verifies OTP via `POST /auth/otp/verify` and receives JWT token

2. **Driver Registration**:
   - Authenticated user registers as driver via `POST /drivers/register`

3. **Real-time Communication**:
   - Client establishes WebSocket connection to `/ws` with JWT authentication
   - Client sends/receives various event types for beacon updates, match events, and location tracking

4. **Event Processing**:
   - Service receives events from other services via NATS
   - Service forwards relevant events to connected clients via WebSockets

## Integration Points

- **Location Service**: Receives location updates from WebSocket and forwards them to the Location Service
- **Match Service**: Processes match proposals from the Match Service and forwards them to clients
- **Ride Service**: Processes ride events from the Ride Service and notifies clients about ride status changes