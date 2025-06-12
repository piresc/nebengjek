package utils

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Length 1", 1},
		{"Length 8", 8},
		{"Length 16", 16},
		{"Length 32", 32},
		{"Length 64", 64},
		{"Length 128", 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomString(tt.length)
			
			assert.NoError(t, err, "Should not return an error")
			assert.Equal(t, tt.length, len(result), "Result length should match requested length")
			
			// Verify it's a valid base64 URL-encoded string (truncated)
			for _, char := range result {
				assert.True(t, 
					(char >= 'A' && char <= 'Z') || 
					(char >= 'a' && char <= 'z') || 
					(char >= '0' && char <= '9') || 
					char == '-' || char == '_',
					"Character should be valid base64 URL character")
			}
		})
	}
}

func TestGenerateRandomString_Uniqueness(t *testing.T) {
	t.Run("Multiple calls produce different results", func(t *testing.T) {
		length := 16
		results := make(map[string]bool)
		
		// Generate 100 random strings and check for uniqueness
		for i := 0; i < 100; i++ {
			result, err := GenerateRandomString(length)
			assert.NoError(t, err)
			
			// Check if we've seen this result before
			assert.False(t, results[result], "Generated string should be unique")
			results[result] = true
		}
	})
}

func TestGenerateRandomString_EdgeCases(t *testing.T) {
	t.Run("Zero length", func(t *testing.T) {
		result, err := GenerateRandomString(0)
		assert.NoError(t, err)
		assert.Empty(t, result, "Zero length should return empty string")
	})
}

func TestGenerateRandomHex(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Length 2", 2},
		{"Length 8", 8},
		{"Length 16", 16},
		{"Length 32", 32},
		{"Length 64", 64},
		{"Length 128", 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomHex(tt.length)
			
			assert.NoError(t, err, "Should not return an error")
			assert.Equal(t, tt.length, len(result), "Result length should match requested length")
			
			// Verify it's a valid hex string
			for _, char := range result {
				assert.True(t, 
					(char >= '0' && char <= '9') || 
					(char >= 'a' && char <= 'f'),
					"Character should be valid hex character")
			}
			
			// Verify it can be decoded as hex
			_, err = hex.DecodeString(result)
			assert.NoError(t, err, "Result should be valid hex string")
		})
	}
}

func TestGenerateRandomHex_Uniqueness(t *testing.T) {
	t.Run("Multiple calls produce different results", func(t *testing.T) {
		length := 16
		results := make(map[string]bool)
		
		// Generate 100 random hex strings and check for uniqueness
		for i := 0; i < 100; i++ {
			result, err := GenerateRandomHex(length)
			assert.NoError(t, err)
			
			// Check if we've seen this result before
			assert.False(t, results[result], "Generated hex string should be unique")
			results[result] = true
		}
	})
}

func TestGenerateRandomHex_EdgeCases(t *testing.T) {
	t.Run("Zero length", func(t *testing.T) {
		result, err := GenerateRandomHex(0)
		assert.NoError(t, err)
		assert.Empty(t, result, "Zero length should return empty string")
	})

	t.Run("Odd length", func(t *testing.T) {
		// Note: This will generate length/2 bytes, so odd lengths will be truncated
		result, err := GenerateRandomHex(3)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(result), "Odd length should be truncated to even")
	})
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		// Valid emails
		{"Simple valid email", "test@example.com", true},
		{"Email with subdomain", "user@mail.example.com", true},
		{"Email with numbers", "user123@example123.com", true},
		{"Email with dots in local part", "user.name@example.com", true},
		{"Email with plus", "user+tag@example.com", true},
		{"Email with dash in local part", "user-name@example.com", true},
		{"Email with underscore", "user_name@example.com", true},
		{"Email with percentage", "user%test@example.com", true},
		{"Long TLD", "user@example.museum", true},
		{"Two letter TLD", "user@example.co", true},
		{"Three letter TLD", "user@example.com", true},
		{"Email with dash in domain", "user@ex-ample.com", true},
		{"Email with numbers in domain", "user@example123.com", true},
		
		// Invalid emails
		{"Missing @", "userexample.com", false},
		{"Missing domain", "user@", false},
		{"Missing local part", "@example.com", false},
		{"Missing TLD", "user@example", false},
		{"Double @", "user@@example.com", false},
		{"Space in email", "user @example.com", false},
		{"Space in domain", "user@exam ple.com", false},
		{"Invalid characters", "user@example..com", false},
		{"Starts with dot", ".user@example.com", false},
		{"Ends with dot", "user.@example.com", false},
		{"Empty string", "", false},
		{"Only @", "@", false},
		{"TLD too short", "user@example.c", false},
		{"Domain starts with dash", "user@-example.com", false},
		{"Domain ends with dash", "user@example-.com", false},
		{"Special characters", "user@example!.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEmail(tt.email)
			assert.Equal(t, tt.expected, result, "Email validation result should match expected")
		})
	}
}

func TestIsValidPhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected bool
	}{
		// Valid phone numbers
		{"US format with +1", "+11234567890", true},
		{"International format", "+628111234567", true},
		{"Without country code", "1234567890", true},
		{"Minimum length (10 digits)", "1234567890", true},
		{"Maximum length (15 digits)", "123456789012345", true},
		{"With plus sign", "+1234567890", true},
		{"European format", "+33123456789", true},
		{"Asian format", "+8612345678901", true},
		
		// Invalid phone numbers
		{"Too short (9 digits)", "123456789", false},
		{"Too long (16 digits)", "1234567890123456", false},
		{"Contains letters", "123abc7890", false},
		{"Contains spaces", "123 456 7890", false},
		{"Contains dashes", "123-456-7890", false},
		{"Contains dots", "123.456.7890", false},
		{"Empty string", "", false},
		{"Only plus sign", "+", false},
		{"Special characters", "+123-456-7890", false},
		{"Multiple plus signs", "++1234567890", false},
		{"Plus in middle", "123+4567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPhoneNumber(tt.phone)
			assert.Equal(t, tt.expected, result, "Phone validation result should match expected")
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{"String shorter than max", "hello", 10, "hello"},
		{"String equal to max", "hello", 5, "hello"},
		{"String longer than max", "hello world", 8, "hello..."},
		{"Empty string", "", 5, ""},
		{"Max length 0", "hello", 0, "..."},
		{"Max length 1", "hello", 1, "..."},
		{"Max length 2", "hello", 2, "..."},
		{"Max length 3", "hello", 3, "..."},
		{"Max length 4", "hello", 4, "h..."},
		{"Long string", "This is a very long string that needs to be truncated", 20, "This is a very lo..."},
		{"Unicode characters", "héllo wörld", 8, "héllo..."},
		{"Single character", "a", 5, "a"},
		{"Exactly at ellipsis boundary", "hello", 8, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result, "Truncated string should match expected")
			
			// Verify result length doesn't exceed maxLength
			// For maxLength 0, 1, 2, we return "..." which is 3 characters
			if tt.maxLength > 0 && tt.maxLength > 2 {
				assert.LessOrEqual(t, len([]rune(result)), tt.maxLength, "Result should not exceed max length")
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal string", "hello world", "hello world"},
		{"String with newlines", "hello\nworld", "hello world"},
		{"String with tabs", "hello\tworld", "hello world"},
		{"String with carriage returns", "hello\rworld", "hello world"},
		{"Mixed whitespace", "hello\n\t\rworld", "hello world"},
		{"Multiple spaces", "hello    world", "hello world"},
		{"Leading and trailing spaces", "  hello world  ", "hello world"},
		{"Only whitespace", "   \n\t\r   ", ""},
		{"Empty string", "", ""},
		{"Single word", "hello", "hello"},
		{"Multiple newlines", "hello\n\n\nworld", "hello world"},
		{"Complex mixed", "  hello\n\tworld\r\n  test  ", "hello world test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result, "Sanitized string should match expected")
			
			// Verify no control characters remain
			for _, char := range result {
				assert.False(t, char == '\n' || char == '\t' || char == '\r', 
					"Result should not contain control characters")
			}
			
			// Verify no leading/trailing spaces
			if len(result) > 0 {
				assert.NotEqual(t, ' ', result[0], "Result should not start with space")
				assert.NotEqual(t, ' ', result[len(result)-1], "Result should not end with space")
			}
		})
	}
}



// Benchmark tests
func BenchmarkGenerateRandomString(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateRandomString(32)
	}
}

func BenchmarkGenerateRandomHex(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateRandomHex(32)
	}
}

func BenchmarkIsValidEmail(b *testing.B) {
	email := "test@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidEmail(email)
	}
}

func BenchmarkIsValidPhoneNumber(b *testing.B) {
	phone := "+1234567890"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidPhoneNumber(phone)
	}
}

func BenchmarkTruncate(b *testing.B) {
	text := "This is a long string that needs to be truncated for testing purposes"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Truncate(text, 20)
	}
}