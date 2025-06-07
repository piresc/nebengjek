# New Relic APM Implementation Guide

This document describes the comprehensive New Relic APM instrumentation implementation for the Nebengjek microservices platform.

## Overview

The implementation provides end-to-end tracing from HTTP requests through business logic to database operations, with distributed tracing across service boundaries.

## Architecture

### Components Implemented

1. **New Relic Middleware** (`internal/pkg/middleware/newrelic.go`)
   - Echo framework integration
   - Transaction creation and management
   - Custom attribute helpers
   - Error reporting utilities

2. **Database Helpers** (`internal/pkg/newrelic/helpers.go`)
   - PostgreSQL operation instrumentation
   - Redis operation instrumentation
   - Business logic timing
   - Custom metrics and events

3. **HTTP Client Instrumentation** (`internal/pkg/http/newrelic_client.go`)
   - Distributed tracing headers
   - External segment timing
   - Service-to-service call tracking

4. **Service Integration**
   - All main.go files updated with middleware registration
   - Handler layer transaction propagation
   - Use case layer business logic instrumentation
   - Repository layer database instrumentation
   - Gateway layer HTTP client instrumentation

## Implementation Details

### 1. Middleware Layer

**File**: `internal/pkg/middleware/newrelic.go`

The New Relic middleware:
- Creates transactions for each HTTP request
- Sets web request/response context
- Propagates transaction context through the request lifecycle
- Provides helper functions for custom attributes and error reporting

**Key Functions**:
- `NewNewRelicMiddleware()` - Creates middleware instance
- `TransactionFromContext()` - Extracts transaction from context
- `AddCustomAttribute()` - Adds business context attributes
- `NoticeError()` - Reports errors to New Relic
- `StartSegment()` - Times operations

### 2. Handler Layer

**Example**: `services/users/handler/http/user.go`

Handlers now:
- Extract New Relic transaction context
- Add business-specific attributes (user_id, operation type)
- Propagate context to use cases
- Report errors with full context
- Log with trace correlation

**Custom Attributes Added**:
- `operation` - The business operation being performed
- `user.id` - User identifier for correlation
- `user.role` - User role (passenger/driver)
- `error` - Error flags for filtering

### 3. Use Case Layer

**Example**: `services/users/usecase/user.go`

Use cases now:
- Start business logic segments for timing
- Add operation-specific attributes
- Report validation and business logic errors
- Track success/failure metrics

**Instrumentation**:
- Function-level timing with `StartBusinessLogicSegment()`
- Validation error tracking
- Business metric recording
- Success/failure attribution

### 4. Repository Layer

**Examples**: 
- `services/users/repository/user.go` (PostgreSQL)
- `services/location/repository/location.go` (Redis)

Repositories now:
- Instrument database operations with datastore segments
- Add query-specific attributes (table, operation type)
- Track database performance metrics
- Report database errors with context

**Database Operations Tracked**:
- PostgreSQL: SELECT, INSERT, UPDATE, DELETE operations
- Redis: GEOADD, GEORADIUS, HMSET, HGETALL operations
- Transaction timing and success rates
- Query-specific metadata

### 5. Gateway Layer

**Example**: `services/match/gateway/location.go`

Gateways now:
- Use instrumented HTTP client for external calls
- Add distributed tracing headers
- Track external service performance
- Report service-to-service errors

**External Tracking**:
- HTTP method and URL
- Response status codes
- Request/response timing
- Service dependency mapping

## Configuration

### Environment Variables

Add these to your service configuration files:

```bash
# New Relic Configuration
NEW_RELIC_ENABLED=true
NEW_RELIC_LICENSE_KEY=your_license_key_here
NEW_RELIC_APP_NAME=nebengjek-users-service
NEW_RELIC_LOGS_ENABLED=true
NEW_RELIC_FORWARD_LOGS=true
```

### Service-Specific App Names

- `nebengjek-users-service`
- `nebengjek-match-service`
- `nebengjek-location-service`
- `nebengjek-rides-service`

## Custom Attributes Reference

### Common Attributes

| Attribute | Description | Example |
|-----------|-------------|---------|
| `operation` | Business operation type | `create_user`, `find_nearby_drivers` |
| `user.id` | User identifier | `uuid-string` |
| `driver.id` | Driver identifier | `uuid-string` |
| `ride.id` | Ride identifier | `uuid-string` |
| `match.id` | Match identifier | `uuid-string` |

### Database Attributes

| Attribute | Description | Example |
|-----------|-------------|---------|
| `db.operation` | Database operation | `select`, `insert`, `update` |
| `db.table` | Database table | `users`, `rides` |
| `redis.operation` | Redis operation | `geoadd`, `hmset` |

### HTTP Attributes

| Attribute | Description | Example |
|-----------|-------------|---------|
| `http.method` | HTTP method | `GET`, `POST` |
| `http.url` | Request URL | `/api/users/123` |
| `http.status_code` | Response status | `200`, `404` |
| `http.service` | Target service | `location-service` |

### Business Logic Attributes

| Attribute | Description | Example |
|-----------|-------------|---------|
| `user.role` | User role | `passenger`, `driver` |
| `search.radius_km` | Search radius | `5.0` |
| `drivers.found_count` | Number of drivers found | `3` |
| `validation.error` | Validation error type | `msisdn_invalid` |

## Testing the Implementation

### 1. Verify Middleware Registration

Check application logs for:
```
New Relic middleware registered
```

### 2. Test Transaction Creation

Make HTTP requests and verify in New Relic:
- Transactions appear in APM
- Transaction names follow pattern: `GET /api/users/{id}`
- Response times are recorded

### 3. Verify Custom Attributes

In New Relic APM:
1. Go to APM → Your Service → Transactions
2. Click on a transaction
3. Check "Attributes" tab for custom attributes

### 4. Test Error Reporting

Trigger errors and verify:
- Errors appear in New Relic Error Analytics
- Error context includes custom attributes
- Stack traces are captured

### 5. Verify Database Instrumentation

Check New Relic APM:
- Database queries appear in "Databases" tab
- Query timing is recorded
- Database operations are categorized

### 6. Test Distributed Tracing

Make service-to-service calls:
1. Trigger a flow that calls multiple services
2. In New Relic, go to APM → Distributed Tracing
3. Verify trace spans across services
4. Check that trace context propagates

### 7. Verify External Services

Check external service calls:
- External services appear in "External services" tab
- HTTP client calls are timed
- Service dependencies are mapped

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Transaction Throughput**
   - Requests per minute by service
   - Transaction response times
   - Error rates

2. **Database Performance**
   - Query response times
   - Database connection pool usage
   - Slow query identification

3. **External Service Dependencies**
   - Service-to-service call latency
   - External service error rates
   - Dependency failure impact

4. **Business Metrics**
   - User registration success rates
   - Driver matching performance
   - Location update frequency

### Recommended Alerts

1. **High Error Rate**: > 5% error rate for 5 minutes
2. **Slow Response Time**: > 2 seconds average response time
3. **Database Slow Queries**: > 1 second query time
4. **External Service Failures**: > 10% failure rate

## Custom Dashboards

Create dashboards for:

1. **Service Health Overview**
   - Response times across all services
   - Error rates by service
   - Throughput metrics

2. **Business Operations**
   - User registrations per hour
   - Driver availability metrics
   - Ride matching success rates

3. **Database Performance**
   - Query performance by operation
   - Connection pool utilization
   - Redis operation timing

4. **Service Dependencies**
   - External service call map
   - Inter-service communication health
   - Distributed trace analysis

## Troubleshooting

### Common Issues

1. **Missing Transactions**
   - Verify middleware is registered
   - Check New Relic license key
   - Ensure NEW_RELIC_ENABLED=true

2. **No Custom Attributes**
   - Verify transaction context propagation
   - Check attribute naming conventions
   - Ensure attributes are added before transaction ends

3. **Missing Database Segments**
   - Verify repository instrumentation
   - Check database helper usage
   - Ensure context propagation to repositories

4. **Broken Distributed Tracing**
   - Verify HTTP client instrumentation
   - Check distributed tracing headers
   - Ensure trace context propagation

### Debug Mode

Enable debug logging:
```bash
NEW_RELIC_DEBUG=true
LOG_LEVEL=debug
```

## Performance Impact

The New Relic instrumentation has minimal performance impact:
- < 1ms overhead per transaction
- Minimal memory footprint
- Asynchronous data transmission
- Configurable sampling rates

## Best Practices

1. **Attribute Naming**: Use consistent, descriptive names
2. **Error Context**: Always include relevant business context
3. **Segment Timing**: Time significant operations only
4. **Custom Events**: Use for business-critical events
5. **Alert Tuning**: Set meaningful thresholds based on SLAs

## Next Steps

1. **Custom Events**: Implement business event tracking
2. **Synthetic Monitoring**: Set up API endpoint monitoring
3. **Infrastructure Monitoring**: Add server/container metrics
4. **Log Correlation**: Enhance log-trace correlation
5. **Performance Baselines**: Establish performance benchmarks