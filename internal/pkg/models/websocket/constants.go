package websocket

import "time"

// WebSocket event types
const (
	// Common events
	EventError = "error"

	// Heartbeat events (leveraging coder/websocket built-in ping/pong)
	EventPing      = "ping"
	EventPong      = "pong"
	EventHeartbeat = "heartbeat"

	// User events
	EventBeaconUpdate = "beacon_update"
	EventFinderUpdate = "finder_update"

	// Location events
	EventLocationUpdate = "location_update"

	// Match events
	EventMatchConfirm  = "match_confirm"
	EventMatchRejected = "match_rejected"

	// Ride events
	EventRideStarted      = "ride_started"      // When a ride is created
	EventRidePickup       = "ride_pickup"       // When driver is on the way to pick up passenger
	EventRideArrived      = "ride_arrived"      // When driver indicates arrival
	EventPaymentRequest   = "payment_request"   // When payment request is generated after arrival
	EventPaymentProcessed = "payment_processed" // When payment is processed
	EventRideCompleted    = "ride_completed"    // When ride is completed and payment processed
)

// Heartbeat configuration leveraging coder/websocket capabilities
const (
	HeartbeatInterval = 30 * time.Second  // Send ping every 30 seconds using conn.Ping()
	HeartbeatTimeout  = 90 * time.Second  // Consider connection dead after 90 seconds
	MaxMissedPings    = 3                 // Max consecutive missed pongs (handled automatically)
)

// WebSocket error codes
const (
	ErrorInvalidFormat     = "invalid_format"
	ErrorInvalidBeacon     = "invalid_beacon"
	ErrorInvalidLocation   = "invalid_location"
	ErrorMatchUpdateFailed = "match_update_failed"
	ErrorUnauthorized      = "unauthorized"
	ErrorSystemUnavailable = "system_unavailable"
	ErrorAccessDenied      = "access_denied"
)

// Error severity levels for WebSocket error handling
type ErrorSeverity int

const (
	// ErrorSeverityClient - Show detailed error to client (validation errors, user input issues)
	ErrorSeverityClient ErrorSeverity = iota
	// ErrorSeverityServer - Hide details from client, log server-side (system errors, database issues)
	ErrorSeverityServer
	// ErrorSeveritySecurity - Minimal info to client + security alert (authentication, authorization)
	ErrorSeveritySecurity
)
