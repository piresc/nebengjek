package entity

type Driver struct {
	ID        string
	Latitude  float64
	Longitude float64
	Status    string
	Distance  float64
}

type Match struct {
	ID              string
	UserID          string
	DriverID        string
	Status          string
	EtaMinutes      float64
	PickupLatitude  float64
	PickupLongitude float64
	DestLatitude    float64
	DestLongitude   float64
}
