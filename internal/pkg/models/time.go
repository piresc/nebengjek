package models

import (
	"time"
)

// Now returns the current time in UTC
func Now() time.Time {
	return time.Now().UTC()
}

// FormatTime formats a time.Time according to RFC3339
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseTime parses a string in RFC3339 format to time.Time
func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// AddDuration adds a duration to a time
func AddDuration(t time.Time, duration time.Duration) time.Time {
	return t.Add(duration)
}

// DurationBetween returns the duration between two times
func DurationBetween(t1, t2 time.Time) time.Duration {
	return t2.Sub(t1)
}
