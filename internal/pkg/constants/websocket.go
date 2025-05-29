package constants

// WebSocket event types
const (
	// Common events
	EventError = "error"

	// User events
	EventBeaconUpdate = "beacon_update"

	// Location events
	EventLocationUpdate = "location_update"

	// Match events
	EventMatchConfirm  = "match_confirm"
	EventMatchRejected = "match_rejected"

	// Ride events
	EventRideCompleted = "ride_completed"
	EventRideArrived   = "ride_arrived"
)

// WebSocket error codes
const (
	ErrorInvalidFormat     = "invalid_format"
	ErrorInvalidBeacon     = "invalid_beacon"
	ErrorInvalidLocation   = "invalid_location"
	ErrorMatchUpdateFailed = "match_update_failed"
)
