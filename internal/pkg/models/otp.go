package models

// OTP represents a one-time password for user authentication
type OTP struct {
	ID     string `json:"id" bson:"_id" db:"id"`
	MSISDN string `json:"msisdn" bson:"msisdn" db:"msisdn"`
	Code   string `json:"code" bson:"code" db:"code"`
}

// LoginRequest represents a request to login with MSISDN
type LoginRequest struct {
	MSISDN string `json:"msisdn" validate:"required"`
}

// VerifyRequest represents a request to verify OTP
type VerifyRequest struct {
	MSISDN string `json:"msisdn" validate:"required"`
	OTP    string `json:"otp" validate:"required"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at"`
}
