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
	EventMatchAccepted = "match_accepted"
	EventMatchRejected = "match_rejected"
	EventMatchExpired  = "match_expired"

	// Trip events
	EventTripStarted   = "trip_started"
	EventTripLocation  = "trip_location"
	EventTripArrived   = "trip_arrived"
	EventTripCompleted = "trip_completed"
	EventTripCancelled = "trip_cancelled"

	// Payment events
	EventPaymentRequest = "payment_request"
	EventPaymentSuccess = "payment_success"
	EventPaymentFailed  = "payment_failed"
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
