package constants

// Redis key formats
const (
	// User Service
	KeyUserOTP = "user:otp:%s" // Format: user:otp:{msisdn}

	// Location Service
	KeyDriverLocation      = "driver:location:%s"    // Format: driver:location:{driver_id}
	KeyPassengerLocation   = "passenger:location:%s" // Format: passenger:location:{passenger_id}
	KeyDriverGeo           = "driver:geo"            // GeoHash set of all driver locations
	KeyPassengerGeo        = "passenger:geo"         // GeoHash set of all passenger locations
	KeyAvailableDrivers    = "drivers:available"     // Set of available driver IDs
	KeyAvailablePassengers = "passengers:available"  // Set of available passenger IDs

	// Match Service
	KeyMatchProposal        = "match:proposal:%s"         // Format: match:proposal:{match_id}
	KeyPassengerMatch       = "passenger:match:%s"        // Format: passenger:match:{passenger_id}
	KeyDriverMatch          = "driver:match:%s"           // Format: driver:match:{driver_id}
	KeyPendingMatchPair     = "match:pending:%s:%s"       // Format: match:pending:{driver_id}:{passenger_id}
	KeyDriverPendingMatches = "driver:pending-matches:%s" // Format: driver:pending-matches:{driver_id}

	// Ride Service
	KeyRideLocation = "rides:location:%s" // Format: trip:location:{trip_id}

	// Active rides tracking - used by match service to prevent matching during active rides
	KeyActiveRideDriver    = "active_ride:driver:%s"    // Format: active_ride:driver:{driver_id} -> ride_id
	KeyActiveRidePassenger = "active_ride:passenger:%s" // Format: active_ride:passenger:{passenger_id} -> ride_id
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
