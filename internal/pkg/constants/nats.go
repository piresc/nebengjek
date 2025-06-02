package constants

// NATS Subjects
const (
	// User Service
	SubjectUserBeacon = "user.beacon"
	SubjectUserFinder = "user.finder"

	// Match Service
	SubjectMatchFound    = "match.found"
	SubjectMatchRejected = "match.rejected"
	SubjectMatchAccepted = "match.accepted"

	// Ride events
	SubjectRidePickup    = "ride.pickup"
	SubjectRideStarted   = "ride.started"
	SubjectRideArrived   = "ride.arrived"
	SubjectRideCompleted = "ride.completed"

	// Location Service
	SubjectLocationUpdate    = "location.update"
	SubjectLocationAggregate = "location.aggregate"
)
