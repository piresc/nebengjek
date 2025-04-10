package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// PREFIXES defines the valid prefixes for different operators
var PREFIXES = struct {
	TELKOMSEL []int
}{
	TELKOMSEL: []int{811, 812, 813, 821, 822, 823, 851, 852, 853},
}

// ValidateMSISDN validates a phone number format and checks if it's a Telkomsel number
func ValidateMSISDN(msisdn string) (bool, string, error) {
	// Clean the input by removing any non-digit characters
	stripped := strings.ReplaceAll(msisdn, "-", "")
	stripped = strings.ReplaceAll(stripped, " ", "")
	stripped = strings.ReplaceAll(stripped, "+", "")

	// Remove country code if present (e.g., 62 for Indonesia)
	if strings.HasPrefix(stripped, "62") {
		stripped = stripped[2:]
	} else if strings.HasPrefix(stripped, "0") {
		stripped = stripped[1:]
	}

	// Validate against Telkomsel prefixes
	prefixesStr := make([]string, len(PREFIXES.TELKOMSEL))
	for i, prefix := range PREFIXES.TELKOMSEL {
		prefixesStr[i] = fmt.Sprintf("%d", prefix)
	}

	// Create pattern for Telkomsel numbers
	pattern := fmt.Sprintf("^(%s)\\d{6,8}$", strings.Join(prefixesStr, "|"))
	isValid := regexp.MustCompile(pattern).MatchString(stripped)

	if !isValid {
		return false, "", fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}

	// Format with country code
	formatted := "62" + stripped

	return true, formatted, nil
}

// GenerateDummyOTP generates a dummy OTP using the last 4 digits of the MSISDN
func GenerateDummyOTP(msisdn string) string {
	// Clean the input
	stripped := strings.ReplaceAll(msisdn, "-", "")
	stripped = strings.ReplaceAll(stripped, " ", "")
	stripped = strings.ReplaceAll(stripped, "+", "")

	// Get last 4 digits
	if len(stripped) >= 4 {
		return stripped[len(stripped)-4:]
	}

	// Fallback if MSISDN is too short (shouldn't happen with validation)
	return "1234"
}
