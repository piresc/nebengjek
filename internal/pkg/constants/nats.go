package constants

// NATS Subjects
const (
	// User Service
	SubjectUserBeacon = "user.beacon"

	// Match Service
	SubjectMatchFound    = "match.found"
	SubjectMatchAccepted = "match.accepted" // Driver accepts initial match proposal
	SubjectMatchConfirm  = "match.confirmed"  // Match service confirms final accepted match (after customer confirmation)
	SubjectMatchRejected = "match.rejected" // Match service signals a rejected match (either by driver or customer)
	SubjectMatchPendingCustomerConfirmation = "match.pending_customer_confirmation" // Match service informs User service (passenger)
	SubjectCustomerMatchConfirmed = "customer.match.confirmed" // User service (passenger) informs Match service
	SubjectCustomerMatchRejected  = "customer.match.rejected"  // User service (passenger) informs Match service

	// Ride events
	SubjectRideStarted   = "ride.started"
	SubjectRideArrived   = "ride.arrived"
	SubjectRideCompleted = "ride.completed"

	// Location Service
	SubjectLocationUpdate    = "location.update"
	SubjectLocationAggregate = "location.aggregate"
)
