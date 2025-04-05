package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// GenerateRandomString generates a random string of the specified length
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateRandomHex generates a random hex string of the specified length
func GenerateRandomHex(length int) (string, error) {
	bytes := make([]byte, length/2)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// IsValidEmail checks if a string is a valid email address
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidPhoneNumber checks if a string is a valid phone number
func IsValidPhoneNumber(phone string) bool {
	// This is a simple regex for international phone numbers
	// It can be adjusted based on specific requirements
	phoneRegex := regexp.MustCompile(`^\+?[0-9]{10,15}$`)
	return phoneRegex.MatchString(phone)
}

// Truncate truncates a string to the specified length and adds ellipsis if needed
func Truncate(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

// SanitizeString removes unwanted characters from a string
func SanitizeString(s string) string {
	// Remove control characters and trim spaces
	return strings.TrimSpace(regexp.MustCompile(`[\p{Cc}\p{Cf}\p{Co}\p{Cs}]`).ReplaceAllString(s, ""))
}

// MaskString masks a portion of a string (useful for PII)
func MaskString(s string, start, end int, maskChar string) string {
	if start < 0 || end > len(s) || start > end {
		return s
	}

	prefix := s[:start]
	middle := strings.Repeat(maskChar, end-start)
	suffix := s[end:]

	return prefix + middle + suffix
}

// MaskEmail masks the local part of an email address
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	localPart := parts[0]
	domain := parts[1]

	var maskedLocal string
	if len(localPart) <= 2 {
		maskedLocal = localPart
	} else {
		maskedLocal = localPart[:2] + strings.Repeat("*", len(localPart)-2)
	}

	return maskedLocal + "@" + domain
}

// MaskPhoneNumber masks a phone number, keeping only the last 4 digits visible
func MaskPhoneNumber(phone string) string {
	cleanPhone := regexp.MustCompile(`[^0-9]`).ReplaceAllString(phone, "")
	if len(cleanPhone) <= 4 {
		return cleanPhone
	}

	return strings.Repeat("*", len(cleanPhone)-4) + cleanPhone[len(cleanPhone)-4:]
}
