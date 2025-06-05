package logger

import (
	"time"

	"go.uber.org/zap"
)

// Field type alias for better abstraction
type Field = zap.Field

// Field construction functions - abstracts zap implementation
// This allows using logger field functions instead of importing zap directly
// Making it easier to switch logging frameworks without changing client code

// String constructs a field that carries a string value
func String(key, val string) Field {
	return zap.String(key, val)
}

// Err constructs a field that carries an error
func Err(err error) Field {
	return zap.Error(err)
}

// Int constructs a field that carries an int value
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 constructs a field that carries an int64 value
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Uint32 constructs a field that carries a uint32 value
func Uint32(key string, val uint32) Field {
	return zap.Uint32(key, val)
}

// Float64 constructs a field that carries a float64 value
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Bool constructs a field that carries a boolean value
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Any constructs a field that carries an arbitrary value
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Duration constructs a field that carries a time.Duration value
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// ErrorField constructs a field that carries an error (alias for Err for backward compatibility)
func ErrorField(err error) Field {
	return zap.Error(err)
}

// Strings constructs a field that carries a slice of strings
func Strings(key string, val []string) Field {
	return zap.Strings(key, val)
}
