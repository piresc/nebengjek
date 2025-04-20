# Location Service Documentation

## Overview

The Location Service manages real-time geospatial data for drivers and passengers in the NebengJek system. It's responsible for tracking user locations, finding nearby drivers, and supporting the matching algorithm with geospatial queries.

## Handlers

The Location Service implements the following handler types:
1. NATS Handlers - For asynchronous location updates and queries
2. Location Handler - For internal location management

### Location Handler (`services/location/handler/location.go`)

The Location Handler provides an interface for location-related operations internally within the service. It doesn't expose HTTP endpoints but serves as an abstraction layer between the location usecase and other components.

**Implementation Details:**
- It's initialized with a reference to the location usecase and configuration.
- Acts as a mediator for location service operations.

### NATS Handler (`services/location/handler/nats.go`)

| Subject | Handler | Purpose |
|---------|---------|---------|
| `location.update` | `handleLocationUpdate` | Processes real-time location updates |
| `location.nearby` | `handleNearbyDriversRequest` | Finds drivers near a specific location |

**Implementation Details:**
- Handles location updates published from the User Service
- Processes nearby driver queries from the Match Service
- Updates Redis geo-indexes for fast spatial lookups

## Core Functionality

### Location Tracking

The Location Service maintains real-time information about user locations:

1. **Location Updates**
   - Receives location updates via NATS from the User Service
   - Updates location data in Redis geo-spatial index
   - Stores historical location data in PostgreSQL for analytics

2. **Spatial Queries**
   - Finds nearby drivers within a specified radius
   - Sorts drivers by distance from passenger
   - Filters available drivers based on beacon status

3. **Optimization**
   - Uses Redis GEOADD and GEORADIUS for efficient spatial indexing
   - Caches frequent location queries for improved performance
   - Implements rate limiting for location updates to prevent system overload

## Database Schema

The Location Service interacts with these key database components:

1. **PostgreSQL Schema**
   - `locations` table - Historical location data with PostGIS geometry
   - Fields: `location_id`, `user_id`, `coordinates` (POINT type), `timestamp`, `metadata`

2. **Redis Data Structures**
   - Geo-indexes for active drivers: `geo:drivers`
   - Geo-indexes for active passengers: `geo:passengers`
   - Cached location data with TTL

## Integration Points

- **User Service**: Receives location updates published by the User Service
- **Match Service**: Provides nearby driver data for the matching algorithm
- **Rides Service**: Provides location data for trip tracking and fare calculation

## Data Flow

1. **Location Update Flow**:
   - User Service publishes location update via WebSocket
   - Location Service receives update via NATS subscriber
   - Service updates Redis geo-index and PostgreSQL store
   
2. **Nearby Drivers Query Flow**:
   - Match Service requests nearby drivers for a passenger
   - Location Service queries Redis geo-index within radius
   - Service returns sorted list of nearby available drivers

## Performance Considerations

The Location Service is optimized for:

1. **High write throughput**: Handles thousands of location updates per second
2. **Low-latency reads**: Retrieves nearby drivers in under 50ms
3. **Scalability**: Horizontally scales for increased load
4. **Fault tolerance**: Implements retry mechanisms for NATS communication issues