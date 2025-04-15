package constants

// NATS Subjects
const (
	// User Service
	SubjectUserBeacon  = "user.beacon"
	SubjectUserUpdated = "user.updated"

	// Match Service
	SubjectMatchFound    = "match.found"
	SubjectMatchAccepted = "match.accepted"
	SubjectMatchRejected = "match.rejected"

	// Location Service
	SubjectLocationUpdate  = "location.update"
	SubjectDriverLocation  = "driver.location"
	SubjectDriverAvailable = "driver.availability"

	// Trip Service
	SubjectTripStarted   = "trip.started"
	SubjectTripEnded     = "trip.ended"
	SubjectTripCancelled = "trip.cancelled"
	SubjectTripUpdate    = "trip.update"

	// Payment Service
	SubjectPaymentRequested = "payment.requested"
	SubjectPaymentCompleted = "payment.completed"
	SubjectPaymentFailed    = "payment.failed"
	SubjectRefundRequested  = "refund.requested"
	SubjectRefundCompleted  = "refund.completed"
)
