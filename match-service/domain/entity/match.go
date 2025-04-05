package entity

type Match struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	DriverID        string  `json:"driver_id"`
	Status          string  `json:"status"`
	EtaMinutes      float64 `json:"eta_minutes"`
	PickupLatitude  float64 `json:"pickup_latitude"`
	PickupLongitude float64 `json:"pickup_longitude"`
	DestLatitude    float64 `json:"destination_latitude"`
	DestLongitude   float64 `json:"destination_longitude"`
	CreatedAt       int64   `json:"created_at"`
	UpdatedAt       int64   `json:"updated_at"`
}