package errors

import "errors"

// Authentication and OTP related errors
var (
	ErrInvalidTelkomselNumber = errors.New("invalid MSISDN format or not a Telkomsel number")
	ErrOTPNotFoundOrExpired   = errors.New("OTP not found or expired")
	ErrInvalidOTP             = errors.New("invalid OTP")
	ErrOTPAlreadyVerified     = errors.New("OTP already verified")
)

// User related errors
var (
	ErrUserNotFound           = errors.New("user not found")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrInvalidUserData        = errors.New("invalid user data")
	ErrUserRegistrationFailed = errors.New("user registration failed")
)

// Driver related errors
var (
	ErrDriverNotFound         = errors.New("driver not found")
	ErrDriverInfoNotFound     = errors.New("driver info not found")
	ErrDriverUpdateFailed     = errors.New("driver update failed")
	ErrInvalidDriverData      = errors.New("invalid driver data")
)

// Database related errors
var (
	ErrDatabaseConnection = errors.New("database connection error")
	ErrDatabaseQuery      = errors.New("database query error")
	ErrTransactionFailed  = errors.New("database transaction failed")
)

// Cache related errors
var (
	ErrCacheConnection = errors.New("cache connection error")
	ErrCacheOperation  = errors.New("cache operation failed")
)