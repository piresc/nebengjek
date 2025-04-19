package constants

// NATS Subjects
const (
	// User Service
	SubjectUserBeacon  = "user.beacon"
	SubjectUserUpdated = "user.updated"

	// Match Service
	SubjectMatchFound    = "match.found"
	SubjectMatchAccepted = "match.accepted"
	SubjectMatchConfirm  = "match.confirmed"
	SubjectMatchRejected = "match.rejected"

	SubjectRideStarted   = "ride.started"
	SubjectRideCompleted = "ride.completed"

	// Location Service
	SubjectLocationUpdate    = "location.update"
	SubjectLocationAggregate = "location.aggregate"
	SubjectDriverLocation    = "driver.location"
	SubjectDriverAvailable   = "driver.availability"
)
