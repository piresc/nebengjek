package constants

// WebSocket event types
const (
	// Common events
	EventError = "error"

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

// WebSocket error codes
const (
	ErrorInvalidFormat     = "invalid_format"
	ErrorInvalidBeacon     = "invalid_beacon"
	ErrorInvalidLocation   = "invalid_location"
	ErrorMatchUpdateFailed = "match_update_failed"
)
