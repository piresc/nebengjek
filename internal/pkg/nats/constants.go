package nats

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

	// WebSocket Multi-Gateway Broadcasting Subjects
	SubjectWSBroadcastUser = "WS.broadcast.user"
	SubjectWSBroadcastRole = "WS.broadcast.role"
	SubjectWSBroadcastAll  = "WS.broadcast.all"
	
	// Cross-server user routing
	SubjectWSRouteUser = "WS.route.user"
	
	// Server status management
	SubjectWSServerRegister   = "WS.server.register"
	SubjectWSServerUnregister = "WS.server.unregister"
	SubjectWSServerHeartbeat  = "WS.server.heartbeat"
)

// NATS Stream Names
const (
	StreamWebSocket = "WEBSOCKET_STREAM"
	StreamWSServer  = "WS_SERVER_STREAM"
)
