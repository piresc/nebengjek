# NebengJek User Flow & Service Interactions

This document details the implementation of user flows and service interactions for the NebengJek trip-hailing platform, focusing on communication protocols (HTTP, WebSocket, NATS) between microservices.

## Communication Flow Diagram

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│  User       │         │  Match      │         │  Location   │         │  Trip       │         │  Payment    │
│  Service    │         │  Service    │         │  Service    │         │  Service    │         │  Service    │
└──────┬──────┘         └──────┬──────┘         └──────┬──────┘         └──────┬──────┘         └──────┬──────┘
       │                       │                       │                       │                       │
       │                       │                       │                       │                       │
       │                       │                       │                       │                       │
┌──────┴──────┐         ┌──────┴──────┐         ┌──────┴──────┐         ┌──────┴──────┐         ┌──────┴──────┐
│ HTTP         │         │ NATS         │         │ WebSocket    │         │ NATS         │         │ NATS         │
│ Login/Beacon │◄────────┤ Match Events │◄────────┤ GPS Updates  │◄────────┤ Trip Events  │◄────────┤ Payment      │
└──────────────┘         └──────────────┘         └──────────────┘         └──────────────┘         └──────────────┘
```

## 1. Login & Beacon Activation

### User/Driver Actions
- Login via User Service (HTTP)
- Toggle availability beacon (HTTP)

### Implementation Details

#### HTTP Endpoints

```go
// User Service - Auth Handler
func (h *AuthHandler) Login(c echo.Context) error {
    // Parse login request
    var req LoginRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request"})
    }
    
    // Validate MSISDN and generate OTP
    otp, err := h.authUsecase.GenerateOTP(req.MSISDN)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: err.Error()})
    }
    
    // In production, send OTP via SMS
    // For development, return OTP in response
    return c.JSON(http.StatusOK, LoginResponse{Message: "OTP sent", OTP: otp})
}

// User Service - User Handler
func (h *UserHandler) ToggleBeacon(c echo.Context) error {
    userID := c.Param("id")
    
    // Parse request
    var req BeaconRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request"})
    }
    
    // Update user availability in database
    err := h.userUsecase.UpdateBeaconStatus(userID, req.IsActive)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: err.Error()})
    }
    
    // Publish beacon event to NATS
    beaconEvent := BeaconEvent{
        UserID:    userID,
        IsActive:  req.IsActive,
        UserType:  req.UserType, // "driver" or "customer"
        Timestamp: time.Now(),
    }
    
    err = h.natsProducer.Publish("user.beacon", beaconEvent)
    if err != nil {
        log.Printf("Failed to publish beacon event: %v", err)
        // Continue execution - don't fail the HTTP request due to NATS issue
    }
    
    return c.JSON(http.StatusOK, BeaconResponse{Message: "Beacon status updated"})
}
```

## 2. Match Driver & Customer

### Match Service Implementation

```go
// Match Service - NATS Consumer Setup
func SetupNATSConsumers(matchUsecase usecase.MatchUsecase, natsAddress string) {
    // Create NATS consumer for beacon events
    beaconConsumer, err := nats.NewConsumer("user.beacon", "match-service", natsAddress, func(data []byte) error {
        var event BeaconEvent
        if err := json.Unmarshal(data, &event); err != nil {
            return err
        }
        
        // Process beacon event
        if event.IsActive {
            // User is now available - add to Redis geospatial index
            if event.UserType == "driver" {
                return matchUsecase.AddAvailableDriver(event.UserID, event.Location)
            } else {
                return matchUsecase.AddAvailableCustomer(event.UserID, event.Location)
            }
        } else {
            // User is no longer available - remove from Redis
            if event.UserType == "driver" {
                return matchUsecase.RemoveAvailableDriver(event.UserID)
            } else {
                return matchUsecase.RemoveAvailableCustomer(event.UserID)
            }
        }
    })
    
    if err != nil {
        log.Fatalf("Failed to setup NATS consumer: %v", err)
    }
    
    // Start background worker to find matches
    go findMatches(matchUsecase, natsProducer)
}

// Background worker to find matches
func findMatches(matchUsecase usecase.MatchUsecase, natsProducer *nats.Producer) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        // Find customers with active beacons
        customers, err := matchUsecase.GetAvailableCustomers()
        if err != nil {
            log.Printf("Error getting available customers: %v", err)
            continue
        }
        
        for _, customer := range customers {
            // Find drivers within 1km radius
            drivers, err := matchUsecase.FindNearbyDrivers(customer.ID, customer.Location, 1.0)
            if err != nil {
                log.Printf("Error finding nearby drivers: %v", err)
                continue
            }
            
            if len(drivers) > 0 {
                // Create match in database
                match, err := matchUsecase.CreateMatch(customer.ID, drivers)
                if err != nil {
                    log.Printf("Error creating match: %v", err)
                    continue
                }
                
                // Publish match found event
                matchEvent := MatchFoundEvent{
                    MatchID:    match.ID,
                    CustomerID: customer.ID,
                    Drivers:    drivers,
                    Timestamp:  time.Now(),
                }
                
                err = natsProducer.Publish("match.found", matchEvent)
                if err != nil {
                    log.Printf("Error publishing match event: %v", err)
                }
            }
        }
    }
}
```

### Redis Geospatial Implementation

```go
// Match Repository - Redis Implementation
func (r *matchRepository) AddAvailableDriver(driverID string, location GeoLocation) error {
    // Add driver to Redis geospatial index
    _, err := r.redisClient.GeoAdd("available_drivers", &redis.GeoLocation{
        Name:      driverID,
        Longitude: location.Longitude,
        Latitude:  location.Latitude,
    }).Result()
    return err
}

func (r *matchRepository) FindNearbyDrivers(customerID string, location GeoLocation, radiusKm float64) ([]Driver, error) {
    // Find drivers within radius using GEORADIUS
    geoOptions := &redis.GeoRadiusQuery{
        Radius:      radiusKm,
        Unit:        "km",
        WithCoord:   true,
        WithDist:    true,
        WithGeoHash: true,
        Count:       10,
        Sort:        "ASC",
    }
    
    results, err := r.redisClient.GeoRadius("available_drivers", location.Longitude, location.Latitude, geoOptions).Result()
    if err != nil {
        return nil, err
    }
    
    var drivers []Driver
    for _, result := range results {
        drivers = append(drivers, Driver{
            ID:       result.Name,
            Distance: result.Dist,
            Location: GeoLocation{
                Latitude:  result.Latitude,
                Longitude: result.Longitude,
            },
        })
    }
    
    return drivers, nil
}
```

## 3. Driver Confirms Match

### HTTP Endpoint

```go
// Match Service - Match Handler
func (h *MatchHandler) ConfirmMatch(c echo.Context) error {
    matchID := c.Param("id")
    driverID := c.Get("user_id").(string) // From JWT token
    
    // Parse request
    var req ConfirmMatchRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request"})
    }
    
    // Update match status in database
    err := h.matchUsecase.ConfirmMatch(matchID, driverID, req.Confirmed)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: err.Error()})
    }
    
    if req.Confirmed {
        // Get match details
        match, err := h.matchUsecase.GetMatch(matchID)
        if err != nil {
            return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: err.Error()})
        }
        
        // Publish match confirmed event
        matchEvent := MatchConfirmedEvent{
            MatchID:    matchID,
            CustomerID: match.CustomerID,
            DriverID:   driverID,
            Timestamp:  time.Now(),
        }
        
        err = h.natsProducer.Publish("match.confirmed", matchEvent)
        if err != nil {
            log.Printf("Failed to publish match confirmed event: %v", err)
            // Continue execution - don't fail the HTTP request due to NATS issue
        }
    }
    
    return c.JSON(http.StatusOK, ConfirmMatchResponse{Message: "Match status updated"})
}
```

## 4. Start Trip & Track GPS

### Trip Service - NATS Consumer

```go
// Trip Service - NATS Consumer Setup
func SetupNATSConsumers(tripUsecase usecase.TripUsecase, natsProducer *nats.Producer, natsAddress string) {
    // Create NATS consumer for match confirmed events
    matchConsumer, err := nats.NewConsumer("match.confirmed", "trip-service", natsAddress, func(data []byte) error {
        var event MatchConfirmedEvent
        if err := json.Unmarshal(data, &event); err != nil {
            return err
        }
        
        // Create new trip
        trip, err := tripUsecase.CreateTrip(event.MatchID, event.CustomerID, event.DriverID)
        if err != nil {
            return err
        }
        
        // Publish trip start event
        tripEvent := triptartEvent{
            TripID:     trip.ID,
            MatchID:    event.MatchID,
            CustomerID: event.CustomerID,
            DriverID:   event.DriverID,
            StartTime:  time.Now(),
        }
        
        return natsProducer.Publish("trip.start", tripEvent)
    })
    
    if err != nil {
        log.Fatalf("Failed to setup NATS consumer: %v", err)
    }
    
    // Create NATS consumer for location updates
    locationConsumer, err := nats.NewConsumer("location.update", "trip-service", natsAddress, func(data []byte) error {
        var event LocationUpdateEvent
        if err := json.Unmarshal(data, &event); err != nil {
            return err
        }
        
        // Process location update for active trip
        return tripUsecase.ProcessLocationUpdate(event.TripID, event.UserID, event.UserType, event.Location)
    })
    
    if err != nil {
        log.Fatalf("Failed to setup NATS consumer: %v", err)
    }
}
```

### WebSocket Implementation for Location Updates

```go
// Location Service - WebSocket Handler
func (h *LocationHandler) HandleWebSocket(c echo.Context) error {
    userID := c.Get("user_id").(string) // From JWT token
    userType := c.Get("user_type").(string) // "driver" or "customer"
    tripID := c.QueryParam("trip_id")
    
    // Upgrade HTTP connection to WebSocket
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true // Allow all origins in development
        },
    }
    
    ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
    if err != nil {
        return err
    }
    defer ws.Close()
    
    // Register client
    client := &WebSocketClient{
        UserID:   userID,
        UserType: userType,
        TripID:   tripID,
        Conn:     ws,
    }
    
    h.locationService.RegisterClient(client)
    defer h.locationService.UnregisterClient(client)
    
    // Process incoming messages
    for {
        _, msg, err := ws.ReadMessage()
        if err != nil {
            break // Client disconnected
        }
        
        // Parse location update
        var locationUpdate LocationUpdate
        if err := json.Unmarshal(msg, &locationUpdate); err != nil {
            log.Printf("Error parsing location update: %v", err)
            continue
        }
        
        // Store location in Redis
        err = h.locationUsecase.UpdateLocation(userID, userType, locationUpdate.Location)
        if err != nil {
            log.Printf("Error updating location: %v", err)
            continue
        }
        
        // If part of active trip, publish location update event
        if tripID != "" {
            locationEvent := LocationUpdateEvent{
                TripID:    tripID,
                UserID:    userID,
                UserType:  userType,
                Location:  locationUpdate.Location,
                Timestamp: time.Now(),
            }
            
            err = h.natsProducer.Publish("location.update", locationEvent)
            if err != nil {
                log.Printf("Error publishing location update: %v", err)
            }
        }
    }
    
    return nil
}
```

## 5. Billing Calculation

```go
// Trip Service - Process Location Updates
func (u *tripUsecase) ProcessLocationUpdate(tripID, userID, userType string, location GeoLocation) error {
    // Only process driver location updates for billing
    if userType != "driver" {
        return nil
    }
    
    // Get trip details
    trip, err := u.tripRepo.GetTrip(tripID)
    if err != nil {
        return err
    }
    
    // Check if trip is active
    if trip.Status != "active" {
        return nil
    }
    
    // Get last billed location
    lastLocation, err := u.tripRepo.GetLastBilledLocation(tripID)
    if err != nil {
        // If no last location, use current as first point
        u.tripRepo.UpdateLastBilledLocation(tripID, location)
        return nil
    }
    
    // Calculate distance using Haversine formula
    distance := calculateDistance(lastLocation, location)
    
    // Only bill if distance >= 1km
    if distance >= 1.0 {
        // Calculate fare: 3000 IDR per km
        fare := int64(distance * 3000)
        
        // Update trip cost
        err = u.tripRepo.IncrementTripCost(tripID, fare)
        if err != nil {
            return err
        }
        
        // Update last billed location
        err = u.tripRepo.UpdateLastBilledLocation(tripID, location)
        if err != nil {
            return err
        }
        
        // Get updated trip total
        updatedTrip, err := u.tripRepo.GetTrip(tripID)
        if err != nil {
            return err
        }
        
        // Publish billing update event
        billingEvent := BillingUpdateEvent{
            TripID:       tripID,
            Distance:     distance,
            Fare:         fare,
            TotalCost:    updatedTrip.TotalCost,
            Timestamp:    time.Now(),
        }
        
        return u.natsProducer.Publish("billing.update", billingEvent)
    }
    
    return nil
}

// Helper function to calculate distance using Haversine formula
func calculateDistance(loc1, loc2 GeoLocation) float64 {
    // Implementation of Haversine formula
    // ...
    return distance // in kilometers
}
```

## 6. End Trip & Payment

### HTTP Endpoint

```go
// Trip Service - Trip Handler
func (h *TripHandler) EndTrip(c echo.Context) error {
    tripID := c.Param("id")
    userID := c.Get("user_id").(string) // From JWT token
    userType := c.Get("user_type").(string)
    
    // Parse request
    var req EndTripRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, ErrorResponse{Message: "Invalid request"})
    }
    
    // Verify user is part of this trip
    trip, err := h.tripUsecase.GetTrip(tripID)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: err.Error()})
    }
    
    if trip.DriverID != userID && trip.CustomerID != userID {
        return c.JSON(http.StatusForbidden, ErrorResponse{Message: "Not authorized"})
    }
    
    // If driver is ending trip, they can adjust the fare
    var adjustmentFactor float64 = 1.0 // Default: 100%
    if userType == "driver" && req.AdjustmentFactor > 0 && req.AdjustmentFactor <= 1.0 {
        adjustmentFactor = req.AdjustmentFactor
    }
    
    // End trip
    finalCost, err := h.tripUsecase.EndTrip(tripID, adjustmentFactor)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, ErrorResponse{Message: err.Error()})
    }
    
    // Publish trip end event
    tripEndEvent := TripEndEvent{
        TripID:           tripID,
        CustomerID:       trip.CustomerID,
        DriverID:         trip.DriverID,
        FinalCost:        finalCost,
        AdjustmentFactor: adjustmentFactor,
        EndTime:          time.Now(),
    }
    
    err = h.natsProducer.Publish("trip.end", tripEndEvent)
    if err != nil {
        log.Printf("Failed to publish trip end event: %v", err)
        // Continue execution - don't fail the HTTP request due to NATS issue
    }
    
    return c.JSON(http.StatusOK, EndTripResponse{
        Message:   "Trip ended successfully",
        FinalCost: finalCost,
    })
}
```

### Payment Service - NATS Consumer

```go
// Payment Service - NATS Consumer Setup
func SetupNATSConsumers(paymentUsecase usecase.PaymentUsecase, natsAddress string) {
    // Create NATS consumer for trip end events
    tripEndConsumer, err := nats.NewConsumer("trip.end", "payment-service", natsAddress, func(data []byte) error {
        var event TripEndEvent
        if err := json.Unmarshal(data, &event); err != nil {
            return err
        }
        
        // Process payment
        // Calculate admin fee (5%)
        adminFee := int64(float64(event.FinalCost) * 0.05)
        driverAmount := event.FinalCost - adminFee
        
        // Create payment record
        payment := Payment{
            TripID:       event.TripID,
            CustomerID:   event.CustomerID,
            DriverID:     event.DriverID,
            TotalAmount:  event.FinalCost,
            AdminFee:     adminFee,
            DriverAmount: driverAmount,
            Status:       "completed",
            Timestamp:    time.Now(),
        }
        
        // Save payment record
        err := paymentUsecase.CreatePayment(payment)
        if err != nil {
            return err
        }
        
        // In a real system, this would integrate with Telkomsel's payment API
        // For now, just log the payment
        log.Printf("Payment processed: Trip %s, Total: %d IDR, Admin Fee: %d IDR, Driver: %d IDR",
            event.TripID, event.FinalCost, adminFee, driverAmount)
        
        return nil
    })
    
    if err != nil {
        log.Fatalf("Failed to setup NATS consumer: %v", err)
    }
}
```

## Protocols & Justification

| **Step**               | **Protocol** | **Reason**                                                                 |
|-------------------------|--------------|----------------------------------------------------------------------------|
| Login/Beacon Toggle     | HTTP         | Synchronous, requires immediate response (e.g., success/failure).         |
| Match Alerts            | WebSocket    | Real-time push to users (no polling).                                      |
| GPS Updates             | WebSocket    | Low-latency, bidirectional for frequent location data.                     |
| Service-to-Service      | NATS         | Decoupled, event-driven communication (e.g., `trip.start`, `location.update`). |

## Implementation Steps

1. **Update User Service**:
   - Add beacon toggle endpoint
   - Integrate NATS producer for beacon events

2. **Enhance Match Service**:
   - Implement NATS consumer for beacon events
   - Add Redis geospatial queries for nearby drivers
   - Create match confirmation endpoint

3. **Develop Location Service**:
   - Implement WebSocket handler for real-time GPS updates
   - Add Redis storage for current locations
   - Create NATS producer for location updates

4. **Build Trip Service**:
   - Implement NATS consumer for match confirmed events
   - Add trip creation and management logic
   - Implement billing calculation based on distance
   - Create trip end endpoint

5. **Create Payment Service**:
   - Implement NATS consumer for trip end events
   - Add payment processing logic with admin fee calculation

6. **Testing**:
   - Test each service individually
   - Perform integration testing of the complete flow
   - Verify correct event propagation between services