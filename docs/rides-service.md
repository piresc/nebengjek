# Rides Service Documentation

## Overview

The Rides Service manages the complete lifecycle of rides in the NebengJek system. It handles ride creation, tracking, billing, fare calculation, and payment processing. This service is activated once a match between a driver and passenger has been confirmed.

## Handlers

The Rides Service implements the following handler types:
1. NATS Handlers - For asynchronous event processing of ride-related events

### NATS Handler (`services/rides/handler/nats.go`)

| Subject | Handler | Purpose |
|---------|---------|---------|
| `match.accept` | `handleMatchAccept` | Creates a new ride when a match is accepted |
| `location.update` | `handleLocationUpdate` | Processes location updates during rides |
| `ride.arrived` | `handleRideArrived` | Processes ride completion when driver arrives at destination |

**Implementation Details:**
- `handleMatchAccept`: Creates a new ride when a driver accepts a match
  - Creates ride entry in the database
  - Initializes billing ledger for the ride
  - Publishes ride.started event to notify users

- `handleLocationUpdate`: Processes location updates during the ride
  - Calculates incremental distance and fare
  - Updates the billing ledger with new entries
  - Maintains ride trajectory for audit purposes

- `handleRideArrived`: Processes ride completion when driver arrives
  - Marks ride as completed in the database
  - Applies any adjustment factors to the final fare
  - Processes payment with appropriate fee distribution
  - Publishes ride.completed event with payment details

## Core Functionality

### Ride Lifecycle Management

The Rides Service manages the ride through its complete lifecycle:

1. **Ride Creation**
   - Creates ride record when a match is accepted
   - Initializes ride status as "pending"
   - Prepares billing structures for the ride

2. **Active Ride Tracking**
   - Tracks driver location updates during the ride
   - Calculates distance traveled in real-time
   - Updates the billing ledger with distance-based charges
   - Updates ride status to "ongoing" when ride starts

3. **Ride Completion**
   - Processes driver's arrival notification
   - Calculates final fare with any adjustments
   - Creates payment record with fee breakdown
   - Updates ride status to "completed"

### Billing and Payment

The service handles the financial aspects of each ride:

1. **Fare Calculation**
   - Base rate of 3000 IDR per kilometer
   - Incremental billing based on distance traveled
   - Support for fare adjustments at completion

2. **Payment Processing**
   - Processes payment when ride completes
   - Calculates admin fee (5% of total fare)
   - Calculates driver payout (95% of total fare)
   - Creates detailed payment record for accounting

## Database Schema

The Rides Service works with these key database components:

1. **PostgreSQL Tables**
   - `rides` - Core ride information
     - Fields: `ride_id`, `driver_id`, `customer_id`, `status`, `total_cost`, `created_at`, `updated_at`
   
   - `billing_ledger` - Distance and cost entries throughout the ride
     - Fields: `entry_id`, `ride_id`, `distance`, `cost`, `created_at`
   
   - `payments` - Payment records for completed rides
     - Fields: `payment_id`, `ride_id`, `adjusted_cost`, `admin_fee`, `driver_payout`, `created_at`

## Integration Points

- **User Service**: Notifies users about ride start/completion events
- **Location Service**: Receives location updates for fare calculation
- **Match Service**: Receives match acceptance to create new rides

## Data Flow

1. **Ride Creation Flow**:
   - Match Service publishes match.accept event
   - Rides Service creates a new ride record
   - Rides Service publishes ride.started event
   - User Service notifies driver and passenger via WebSocket

2. **Ride Tracking Flow**:
   - User Service receives location updates via WebSocket
   - User Service publishes location.update events to NATS
   - Rides Service processes updates and calculates fares

3. **Ride Completion Flow**:
   - Driver signals arrival via WebSocket
   - Rides Service processes completion and calculates final payment
   - Rides Service publishes ride.completed event with payment details
   - User Service notifies both users about completion and payment

## Performance Considerations

The Rides Service is designed for:

1. **Scalability**: Horizontally scales to support thousands of concurrent rides
2. **Data Integrity**: Ensures consistent billing even with communication disruptions
3. **Real-time processing**: Updates fare calculations with minimal latency
4. **Audit trail**: Maintains detailed records for financial reconciliation