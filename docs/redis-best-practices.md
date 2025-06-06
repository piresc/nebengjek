# Redis Best Practices

## Overview

This document outlines the Redis best practices implemented in our ride-sharing platform to ensure optimal performance, memory management, and data consistency.

## Core Principle: Always Use TTL

**🚨 CRITICAL RULE: Every Redis key MUST have a TTL (Time To Live) set to prevent memory leaks and ensure data freshness.**

### Why TTL is Essential

1. **Memory Management**: Prevents indefinite accumulation of stale data
2. **Cost Optimization**: Reduces Redis memory usage and associated costs
3. **Data Freshness**: Ensures only relevant, recent data is available
4. **System Reliability**: Prevents Redis from running out of memory
5. **Automatic Cleanup**: Eliminates need for manual data cleanup processes

## TTL Implementation Patterns

### 1. Set TTL During Key Creation

**✅ CORRECT - Set TTL with initial value:**
```go
// Set key with TTL in one operation
err := redisClient.Set(ctx, key, value, 30*time.Minute)
```

**❌ INCORRECT - Setting key without TTL:**
```go
// This creates a key that persists forever
err := redisClient.Set(ctx, key, value, 0)
```

### 2. Set TTL After Key Creation

**✅ CORRECT - Set TTL immediately after creation:**
```go
// Create key
err := redisClient.HMSet(ctx, key, data)
if err != nil {
    return err
}

// Immediately set TTL
err = redisClient.Expire(ctx, key, 30*time.Minute)
if err != nil {
    return err
}
```

### 3. Configurable TTL Values

**✅ CORRECT - Use configuration for TTL values:**
```go
type Config struct {
    AvailabilityTTLMinutes int `json:"availability_ttl_minutes"`
    ActiveRideTTLHours     int `json:"active_ride_ttl_hours"`
}

// Use configured TTL
ttl := time.Duration(config.AvailabilityTTLMinutes) * time.Minute
err := redisClient.Set(ctx, key, value, ttl)
```

## Current Implementation

### ✅ Properly Configured Keys

#### 1. User Availability (Location Service)
- **Keys**: `driver_geo`, `available_drivers`, `passenger_geo`, `available_passengers`, `driver_location:{id}`, `passenger_location:{id}`
- **TTL**: Configurable via `LOCATION_AVAILABILITY_TTL_MINUTES` (default: 30 minutes)
- **Purpose**: Automatic expiration of user availability to prevent stale location data
- **Implementation**: [`services/location/repository/location.go`](services/location/repository/location.go)

#### 2. Active Ride Tracking (Match Service)
- **Keys**: `active_ride:driver:{driverID}`, `active_ride:passenger:{passengerID}`
- **TTL**: Configurable via `MATCH_ACTIVE_RIDE_TTL_HOURS` (default: 24 hours)
- **Purpose**: Prevent memory leaks from incomplete rides
- **Implementation**: [`services/match/repository/match.go`](services/match/repository/match.go)

#### 3. OTP Storage (Users Service)
- **Keys**: `user_otp:{msisdn}`
- **TTL**: 5 minutes (hardcoded for security)
- **Purpose**: Automatic expiration of one-time passwords
- **Implementation**: [`services/users/repository/otp.go`](services/users/repository/otp.go)

#### 4. Rate Limiting (Middleware)
- **Keys**: Rate limiting counters
- **TTL**: Configurable period-based TTL
- **Purpose**: Automatic reset of rate limit counters
- **Implementation**: [`internal/pkg/middleware/rate_limiter.go`](internal/pkg/middleware/rate_limiter.go)

#### 5. Ride Location Tracking (Location Service)
- **Keys**: `ride_location:{rideID}`
- **TTL**: 24 hours (for trip history analysis)
- **Purpose**: Cleanup of completed ride location data
- **Implementation**: [`services/location/repository/location.go`](services/location/repository/location.go)

## Configuration Guidelines

### Environment Variables

```bash
# Location Service
LOCATION_AVAILABILITY_TTL_MINUTES=30

# Match Service  
MATCH_ACTIVE_RIDE_TTL_HOURS=24

# Users Service (OTP TTL is hardcoded for security)
# Rate limiting TTL is configured per endpoint
```

### Recommended TTL Values

| Data Type | Recommended TTL | Reasoning |
|-----------|----------------|-----------|
| User Availability | 15-60 minutes | Balance between freshness and performance |
| Active Ride Tracking | 12-48 hours | Covers longest possible ride scenarios |
| OTP Codes | 5-10 minutes | Security requirement |
| Rate Limiting | 1 minute - 1 hour | Based on rate limit window |
| Location History | 24-72 hours | Sufficient for analytics and debugging |
| Session Data | 1-24 hours | Based on user session requirements |

## Implementation Checklist

### For New Redis Operations

- [ ] **TTL Set**: Every new Redis key has a TTL configured
- [ ] **Configurable**: TTL values are configurable via environment variables
- [ ] **Documented**: TTL purpose and duration are documented
- [ ] **Tested**: TTL behavior is covered in unit tests
- [ ] **Monitored**: Key expiration is logged for debugging

### Code Review Checklist

- [ ] **No Zero TTL**: No `Set(ctx, key, value, 0)` calls
- [ ] **Expire After Create**: `Expire()` called immediately after key creation
- [ ] **Error Handling**: TTL setting errors are properly handled
- [ ] **Configuration**: TTL values come from configuration, not hardcoded
- [ ] **Logging**: TTL information is included in relevant log messages

## Monitoring and Debugging

### Key Metrics to Monitor

1. **Memory Usage**: Track Redis memory consumption over time
2. **Key Count**: Monitor total number of keys in Redis
3. **Expiration Rate**: Track how many keys are expiring per minute
4. **TTL Distribution**: Monitor TTL values across different key types

### Debugging Commands

```bash
# Check TTL of a specific key
redis-cli TTL "key_name"

# Find keys without TTL (returns -1)
redis-cli --scan --pattern "*" | xargs -I {} redis-cli TTL {} | grep -B1 "^-1$"

# Monitor key expiration events
redis-cli --latency-history -i 1 expire

# Check memory usage by key pattern
redis-cli --bigkeys
```

### Common Issues and Solutions

#### Issue: Keys Without TTL
**Symptoms**: Gradual memory increase, stale data
**Solution**: Add TTL to all key creation operations

#### Issue: TTL Too Short
**Symptoms**: Frequent cache misses, performance degradation
**Solution**: Increase TTL based on data usage patterns

#### Issue: TTL Too Long
**Symptoms**: Stale data, memory bloat
**Solution**: Decrease TTL to match data freshness requirements

## Testing TTL Implementation

### Unit Test Examples

```go
func TestKeyHasTTL(t *testing.T) {
    // Create key with TTL
    err := repo.SetActiveRide(ctx, driverID, passengerID, rideID)
    assert.NoError(t, err)
    
    // Verify TTL is set
    ttl, err := redisClient.GetClient().TTL(ctx, key).Result()
    assert.NoError(t, err)
    assert.Greater(t, ttl, time.Duration(0))
    assert.LessOrEqual(t, ttl, 24*time.Hour)
}
```

### Integration Test Examples

```go
func TestTTLExpiration(t *testing.T) {
    // Set short TTL for testing
    shortTTL := 100 * time.Millisecond
    err := redisClient.Set(ctx, key, value, shortTTL)
    assert.NoError(t, err)
    
    // Wait for expiration
    time.Sleep(200 * time.Millisecond)
    
    // Verify key is expired
    _, err = redisClient.Get(ctx, key)
    assert.Equal(t, redis.Nil, err)
}
```

## Migration Guide

### For Existing Keys Without TTL

1. **Identify**: Use Redis commands to find keys without TTL
2. **Analyze**: Determine appropriate TTL for each key type
3. **Update**: Modify code to set TTL during key creation
4. **Deploy**: Roll out changes with monitoring
5. **Cleanup**: Manually set TTL on existing keys if needed

### Example Migration Script

```bash
#!/bin/bash
# Set TTL on existing keys (run with caution)

# Set 30-minute TTL on availability keys
redis-cli --scan --pattern "available_*" | xargs -I {} redis-cli EXPIRE {} 1800

# Set 24-hour TTL on active ride keys  
redis-cli --scan --pattern "active_ride:*" | xargs -I {} redis-cli EXPIRE {} 86400
```

## Conclusion

Proper TTL management is crucial for maintaining a healthy Redis instance. By following these best practices, we ensure:

- **Predictable Memory Usage**: No unexpected memory growth
- **Data Freshness**: Only current, relevant data is available
- **System Reliability**: Redis remains stable under load
- **Cost Efficiency**: Optimal resource utilization

**Remember: Every Redis key MUST have a TTL. No exceptions.**