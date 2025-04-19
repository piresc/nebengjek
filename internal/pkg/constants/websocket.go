package constants

// WebSocket event types
const (
	// Common events
	EventError = "error"
	EventPing  = "ping"
	EventPong  = "pong"

	// User events
	EventBeaconUpdate  = "beacon_update"
	EventProfileUpdate = "profile_update"

	// Location events
	EventLocationUpdate = "location_update"
	EventNearbyDrivers  = "nearby_drivers"

	// Match events
	EventMatchFound    = "match_found"
	EventMatchConfirm  = "match_confirm"
	EventMatchAccept   = "match_accept"
	EventMatchRejected = "match_rejected"
	EventMatchExpired  = "match_expired"

	// Ride events
	EventRideCompleted = "ride_completed"
	EventRideArrived   = "ride_arrived"
)

// WebSocket error codes
const (
	ErrorInvalidFormat     = "invalid_format"
	ErrorValidationFailed  = "validation_failed"
	ErrorUnauthorized      = "unauthorized"
	ErrorInternalError     = "internal_error"
	ErrorRateLimitExceeded = "rate_limit_exceeded"
	ErrorInvalidBeacon     = "invalid_beacon"
	ErrorInvalidLocation   = "invalid_location"
	ErrorMatchNotFound     = "match_not_found"
	ErrorTripNotFound      = "trip_not_found"
	ErrorPaymentFailed     = "payment_failed"
	ErrorMatchUpdateFailed = "match_update_failed"
)
