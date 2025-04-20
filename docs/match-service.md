# Match Service Documentation

## Overview

The Match Service is responsible for pairing drivers with passengers in the NebengJek system. It processes beacon events, finds nearby drivers using the Location Service, and manages the match proposal lifecycle.

## Handlers

The Match Service implements the following handler types:
1. NATS Handlers - For asynchronous event processing

### NATS Handler (`services/match/handler/nats.go`)

| Subject | Handler | Purpose |
|---------|---------|---------|
| `user.beacon` | `handleBeaconEvent` | Processes beacon status changes from users |
| `match.request` | `handleMatchRequest` | Handles passenger requests for matches |
| `match.accept` | `handleMatchAccept` | Processes driver's acceptance of a match |
| `match.reject` | `handleMatchReject` | Processes driver's rejection of a match |

**Implementation Details:**
- `handleBeaconEvent`: Updates driver/passenger availability in Redis when their beacon status changes
  - If active: Adds the user's location to appropriate geo-index
  - If inactive: Removes the user from the geo-index
  
- `handleMatchRequest`: Processes passenger requests for a ride
  - Queries Location Service for nearby drivers
  - Creates match proposals and publishes them to drivers
  - Stores match proposals in the database

- `handleMatchAccept`: Processes when a driver accepts a match
  - Updates match status in the database
  - Notifies the passenger of the acceptance
  - Publishes an event for ride creation
  
- `handleMatchReject`: Processes when a driver rejects a match
  - Updates match status in the database
  - Finds the next available driver if possible
  - Notifies the passenger if no drivers are available

## Core Functionality

### Match Algorithm

The Match Service implements the following matching algorithm:

1. **Availability Tracking**
   - Maintains a list of available drivers via beacon events
   - Stores driver locations in a Redis geo-index
   - Updates availability state when drivers accept/reject matches

2. **Match Creation**
   - When a passenger requests a ride, finds nearby drivers within a radius
   - Sorts drivers by proximity and rating
   - Creates a match proposal for the closest available driver

3. **Match Lifecycle Management**
   - Tracks match proposals through states: pending, accepted, rejected, expired
   - Implements timeout logic for match proposals
   - Maintains match history for analytics and dispute resolution

## Database Schema

The Match Service works with these key database components:

1. **PostgreSQL Tables**
   - `matches` - Match proposals and their status
   - Fields: `match_id`, `driver_id`, `passenger_id`, `status`, `created_at`, `updated_at`
   
2. **Redis Data Structures**
   - Active match proposals: `match:proposals:{driverId}`
   - Match expiry trackers: `match:expiry:{matchId}`

## Integration Points

- **User Service**: Receives beacon events and dispatches match events to users
- **Location Service**: Queries for nearby drivers based on passenger location
- **Rides Service**: Triggers ride creation when a match is accepted

## Data Flow

1. **Match Request Flow**:
   - Passenger requests a ride via User Service WebSocket
   - User Service publishes a match request event to NATS
   - Match Service receives request and finds nearby drivers
   - Match Service creates and publishes match proposals
   
2. **Match Acceptance Flow**:
   - Driver accepts match via User Service WebSocket
   - User Service publishes match acceptance to NATS
   - Match Service updates match status and notifies the passenger
   - Match Service publishes event to create a ride

## Performance Considerations

The Match Service is optimized for:

1. **Low-latency matching**: Matches drivers to passengers quickly
2. **Fairness**: Distributes ride requests evenly among drivers
3. **Reliability**: Ensures matching process completes even with communication failures
4. **Concurrency**: Handles multiple match requests simultaneously without conflicts