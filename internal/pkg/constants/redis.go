package constants

// Redis key formats
const (
	// User Service
	KeyUserOTP     = "user:otp:%s"     // Format: user:otp:{msisdn}
	KeyUserSession = "user:session:%s" // Format: user:session:{user_id}
	KeyUserBeacon  = "user:beacon:%s"  // Format: user:beacon:{user_id}

	// Location Service
	KeyDriverLocation   = "driver:location:%s" // Format: driver:location:{driver_id}
	KeyDriverGeo        = "driver:geo"         // GeoHash set of all driver locations
	KeyPassengerGeo     = "passenger:geo"      // GeoHash set of all passenger locations
	KeyAvailableDrivers = "drivers:available"  // Set of available driver IDs
	DriverLocationKey   = "drivers:locations"  // GeoHash set of all driver locations for real-time tracking

	// Match Service
	KeyMatchProposal  = "match:proposal:%s"  // Format: match:proposal:{match_id}
	KeyPassengerMatch = "passenger:match:%s" // Format: passenger:match:{passenger_id}
	KeyDriverMatch    = "driver:match:%s"    // Format: driver:match:{driver_id}

	KeyRideLocation = "rides:location:%s" // Format: trip:location:{trip_id}

	// Rate Limiting
	KeyRateLimit = "rate:limit:%s:%s" // Format: rate:limit:{resource}:{ip}
)

// Redis hash fields
const (
	FieldLatitude    = "lat"
	FieldLongitude   = "lng"
	FieldTimestamp   = "ts"
	FieldStatus      = "status"
	FieldDriverID    = "driver_id"
	FieldPassengerID = "passenger_id"
	FieldPickupLoc   = "pickup_loc"
	FieldDropoffLoc  = "dropoff_loc"
	FieldPrice       = "price"
	FieldDistance    = "distance"
	FieldDuration    = "duration"
)
