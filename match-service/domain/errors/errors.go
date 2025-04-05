package errors

import "errors"

var (
	ErrNoDriversAvailable = errors.New("no available drivers found")
	ErrInvalidUserID      = errors.New("user_id is required")
	ErrInvalidDriverID    = errors.New("driver_id is required")
	ErrMatchNotFound      = errors.New("match not found")
	ErrInvalidLocation    = errors.New("invalid location coordinates")
)
