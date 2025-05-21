package constants

// NATS Subjects
const (
	// User Service
	SubjectUserBeacon = "user.beacon"

	// Match Service
	SubjectMatchFound    = "match.found"
	SubjectMatchAccepted = "match.accepted"
	SubjectMatchConfirm  = "match.confirmed"
	SubjectMatchRejected = "match.rejected"

	// Ride events
	SubjectRideStarted   = "ride.started"
	SubjectRideArrived   = "ride.arrived"
	SubjectRideCompleted = "ride.completed"

	// Location Service
	SubjectLocationUpdate    = "location.update"
	SubjectLocationAggregate = "location.aggregate"
)
