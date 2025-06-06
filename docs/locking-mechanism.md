# Ride Locking Mechanism

## Overview

The ride locking mechanism ensures that drivers and passengers cannot be matched for multiple rides simultaneously during an active ride lifecycle. This document explains how the locking system works and its interaction with the location-based matching system.

## Architecture Principles

### Separation of Concerns

1. **Locking System**: Manages ride state to prevent double-matching
2. **Location System**: Handles geo-indexing based on real-time user events
3. **Matching System**: Creates matches between available users

### Key Design Decisions

- **No Location Dependency**: Locking/unlocking does not require location data
- **Event-Driven**: Users re-enter available pool organically via location events
- **State-Based**: Active ride tracking prevents matching conflicts

## Ride Lifecycle and Locking

### 1. Match Acceptance Phase

When a match is fully accepted by both driver and passenger:

```go
// In match usecase - when both parties confirm
if match.DriverConfirmed && match.PassengerConfirmed {
    match.Status = models.MatchStatusAccepted
    
    // Remove users from available pools (LOCK)
    uc.locationGW.RemoveAvailableDriver(ctx, match.DriverID.String())
    uc.locationGW.RemoveAvailablePassenger(ctx, match.PassengerID.String())
}
```

**Effect**: Users are removed from geo-spatial indexes and cannot be matched for new rides.

### 2. Ride Pickup Phase

When a ride pickup event is received:

```go
// Store active ride information
uc.SetActiveRide(ctx, ridePickup.DriverID, ridePickup.PassengerID, ridePickup.RideID)

// Additional safety locks (redundant removal)
uc.RemoveDriverFromPool(ctx, ridePickup.DriverID)
uc.RemovePassengerFromPool(ctx, ridePickup.PassengerID)
```

**Effect**: Active ride tracking is established in Redis for state management.

### 3. Ride Completion Phase

When a ride completion event is received:

```go
// Remove active ride tracking
uc.RemoveActiveRide(ctx, driverID, passengerID)

// Release from ride lock (NO LOCATION REQUIRED)
uc.ReleaseDriver(driverID)
uc.ReleasePassenger(passengerID)
```

**Effect**: Users are released from ride lock but NOT automatically added back to available pools.

### 4. Re-availability Phase

Users become available for new matches when they actively send location updates:

- **Drivers**: Send beacon events with current location
- **Passengers**: Send finder events with current location and destination

```go
// In beacon/finder event handlers
if event.IsActive {
    // Check if user has active ride before adding to pool
    hasActiveRide, err := uc.HasActiveRide(ctx, event.UserID, isDriver)
    if !hasActiveRide {
        // Add to available pool with current location
        uc.addDriverToPool(ctx, event.UserID, location)
    }
}
```

## Locking Implementation Details

### Active Ride Tracking

**Storage**: Redis key-value pairs
- `active_ride:driver:{driverID}` → `{rideID}`
- `active_ride:passenger:{passengerID}` → `{rideID}`

**Purpose**: Prevent matching users who are already in active rides

### Location Pool Management

**Storage**: Redis geo-spatial indexes
- `available_drivers` → Geo-indexed driver locations
- `available_passengers` → Geo-indexed passenger locations

**Purpose**: Enable efficient nearby user searches for matching

### Lock States

| State | Driver in Geo-Index | Passenger in Geo-Index | Active Ride Tracking |
|-------|-------------------|----------------------|-------------------|
| Available | ✅ | ✅ | ❌ |
| Matched | ❌ | ❌ | ❌ |
| In Ride | ❌ | ❌ | ✅ |
| Completed | ❌ | ❌ | ❌ |

## Error Handling and Recovery

### Previous Issue (Fixed)

**Problem**: Release methods tried to fetch "last known location" from location service to add users back to available pools.

**Why it Failed**: Locations were already deleted when users were removed from available pools during match acceptance.

**Solution**: Release methods now only clear the ride lock without requiring location data.

### Current Approach

1. **Graceful Release**: No location dependency for unlocking
2. **Natural Re-entry**: Users re-enter available pool via organic location events
3. **Self-Healing**: System recovers automatically when users send location updates
4. **No Stale Data**: Only current, real-time locations are used for matching

## Benefits of This Design

### 1. Reliability
- No dependency on stale location data
- Robust error handling during ride completion
- Self-healing system behavior

### 2. Performance
- Efficient geo-spatial queries on active users only
- No unnecessary location API calls during cleanup
- Event-driven architecture reduces coupling

### 3. Data Consistency
- Single source of truth for user availability
- Real-time location data for matching
- Clear state transitions

### 4. Scalability
- Stateless release operations
- Distributed Redis storage
- Event-based communication

## Monitoring and Debugging

### Key Metrics to Monitor

1. **Active Ride Count**: Number of users with active ride tracking
2. **Available Pool Size**: Number of users in geo-spatial indexes
3. **Release Success Rate**: Percentage of successful ride completions
4. **Re-entry Time**: Time between ride completion and re-availability

### Common Issues and Solutions

**Issue**: User stuck in "locked" state after ride completion
**Diagnosis**: Check active ride tracking in Redis
**Solution**: User will become available on next beacon/finder event

**Issue**: User not appearing in matches after ride
**Diagnosis**: Check if user is sending location events
**Solution**: Verify beacon/finder event publishing

## Future Enhancements

### Recent Enhancements

1. **TTL-based Expiration**: Automatic expiration of user availability in pools
   - Configurable TTL for beacon/finder availability
   - Prevents stale location data accumulation
   - Default: 30 minutes (configurable via `LOCATION_AVAILABILITY_TTL_MINUTES`)

2. **Redis TTL Monitoring**: Tool to identify keys without proper TTL
   - Usage: `go run cmd/redis-ttl-checker/main.go config/location.env`
   - Helps prevent memory leaks from persistent keys

### Potential Future Improvements

1. **Timeout-based Release**: Automatically release locks after ride timeout
2. **Health Checks**: Periodic cleanup of stale active ride tracking
3. **Metrics Dashboard**: Real-time monitoring of locking system health
4. **Circuit Breakers**: Fallback mechanisms for Redis failures

### Backward Compatibility

This locking mechanism maintains compatibility with existing:
- NATS event schemas
- HTTP API endpoints
- Database schemas
- Client applications

The changes are purely internal to the match service logic and do not affect external interfaces.