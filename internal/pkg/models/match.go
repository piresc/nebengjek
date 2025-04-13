package models

// MatchFound represents a match found event
type MatchFound struct {
	MSISDNPassenger   string      `json:"msisdn_user"`
	MSISDNDriver      string      `json:"msisdn_driver"`
	PassengerLocation GeoLocation `json:"passenger_location"`
	DriverLocation    GeoLocation `json:"driver_location"`
}
