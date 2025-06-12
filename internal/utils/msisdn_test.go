package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateMSISDN(t *testing.T) {
	tests := []struct {
		name           string
		msisdn         string
		expectValid    bool
		expectedFormat string
		expectError    bool
	}{
		// Valid Telkomsel numbers
		{
			name:           "Valid 811 prefix with country code",
			msisdn:         "628111234567",
			expectValid:    true,
			expectedFormat: "628111234567",
			expectError:    false,
		},
		{
			name:           "Valid 812 prefix with leading zero",
			msisdn:         "08121234567",
			expectValid:    true,
			expectedFormat: "628121234567",
			expectError:    false,
		},
		{
			name:           "Valid 813 prefix without prefix",
			msisdn:         "8131234567",
			expectValid:    true,
			expectedFormat: "628131234567",
			expectError:    false,
		},
		{
			name:           "Valid 821 prefix with spaces",
			msisdn:         "0821 1234 567",
			expectValid:    true,
			expectedFormat: "628211234567",
			expectError:    false,
		},
		{
			name:           "Valid 822 prefix with dashes",
			msisdn:         "0822-1234-567",
			expectValid:    true,
			expectedFormat: "628221234567",
			expectError:    false,
		},
		{
			name:           "Valid 823 prefix with plus sign",
			msisdn:         "+628231234567",
			expectValid:    true,
			expectedFormat: "628231234567",
			expectError:    false,
		},
		{
			name:           "Valid 851 prefix",
			msisdn:         "08511234567",
			expectValid:    true,
			expectedFormat: "628511234567",
			expectError:    false,
		},
		{
			name:           "Valid 852 prefix",
			msisdn:         "08521234567",
			expectValid:    true,
			expectedFormat: "628521234567",
			expectError:    false,
		},
		{
			name:           "Valid 853 prefix",
			msisdn:         "08531234567",
			expectValid:    true,
			expectedFormat: "628531234567",
			expectError:    false,
		},
		{
			name:           "Valid with 8 digits after prefix",
			msisdn:         "081112345678",
			expectValid:    true,
			expectedFormat: "6281112345678",
			expectError:    false,
		},
		{
			name:           "Valid with 6 digits after prefix",
			msisdn:         "0811123456",
			expectValid:    true,
			expectedFormat: "62811123456",
			expectError:    false,
		},
		// Invalid numbers
		{
			name:           "Invalid prefix (not Telkomsel)",
			msisdn:         "08561234567",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Too short",
			msisdn:         "081112345",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Too long",
			msisdn:         "0811123456789",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Empty string",
			msisdn:         "",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Invalid characters",
			msisdn:         "0811abc1234",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Wrong country code",
			msisdn:         "658111234567",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Invalid prefix 814",
			msisdn:         "08141234567",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Invalid prefix 820",
			msisdn:         "08201234567",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Invalid prefix 824",
			msisdn:         "08241234567",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Invalid prefix 850",
			msisdn:         "08501234567",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
		{
			name:           "Invalid prefix 854",
			msisdn:         "08541234567",
			expectValid:    false,
			expectedFormat: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid, formatted, err := ValidateMSISDN(tt.msisdn)

			assert.Equal(t, tt.expectValid, isValid, "Validation result should match expected")
			assert.Equal(t, tt.expectedFormat, formatted, "Formatted result should match expected")

			if tt.expectError {
				assert.Error(t, err, "Should return an error")
			} else {
				assert.NoError(t, err, "Should not return an error")
			}
		})
	}
}

func TestValidateMSISDN_EdgeCases(t *testing.T) {
	t.Run("Multiple spaces and dashes", func(t *testing.T) {
		msisdn := "  +62 - 811 - 1234 - 567  "
		isValid, formatted, err := ValidateMSISDN(msisdn)
		
		assert.True(t, isValid)
		assert.Equal(t, "628111234567", formatted)
		assert.NoError(t, err)
	})

	t.Run("Mixed formatting", func(t *testing.T) {
		msisdn := "+62-812 123 4567"
		isValid, formatted, err := ValidateMSISDN(msisdn)
		
		assert.True(t, isValid)
		assert.Equal(t, "628121234567", formatted)
		assert.NoError(t, err)
	})

	t.Run("Only digits", func(t *testing.T) {
		msisdn := "8131234567"
		isValid, formatted, err := ValidateMSISDN(msisdn)
		
		assert.True(t, isValid)
		assert.Equal(t, "628131234567", formatted)
		assert.NoError(t, err)
	})
}

func TestGenerateDummyOTP(t *testing.T) {
	tests := []struct {
		name           string
		msisdn         string
		expectedLength int
		expectedOTP    string
	}{
		{
			name:           "Standard MSISDN",
			msisdn:         "628111234567",
			expectedLength: 4,
			expectedOTP:    "4567",
		},
		{
			name:           "MSISDN with leading zero",
			msisdn:         "08121234567",
			expectedLength: 4,
			expectedOTP:    "4567",
		},
		{
			name:           "MSISDN ending with zeros",
			msisdn:         "628111230000",
			expectedLength: 4,
			expectedOTP:    "0000",
		},
		{
			name:           "Short MSISDN",
			msisdn:         "62811123456",
			expectedLength: 4,
			expectedOTP:    "3456",
		},
		{
			name:           "MSISDN with 8 digits after prefix",
			msisdn:         "6281112345678",
			expectedLength: 4,
			expectedOTP:    "5678",
		},
		{
			name:           "Very short number",
			msisdn:         "123",
			expectedLength: 3,
			expectedOTP:    "123",
		},
		{
			name:           "Empty string",
			msisdn:         "",
			expectedLength: 0,
			expectedOTP:    "",
		},
		{
			name:           "Single digit",
			msisdn:         "5",
			expectedLength: 1,
			expectedOTP:    "5",
		},
		{
			name:           "Two digits",
			msisdn:         "56",
			expectedLength: 2,
			expectedOTP:    "56",
		},
		{
			name:           "Three digits",
			msisdn:         "567",
			expectedLength: 3,
			expectedOTP:    "567",
		},
		{
			name:           "Exactly four digits",
			msisdn:         "5678",
			expectedLength: 4,
			expectedOTP:    "5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp := GenerateDummyOTP(tt.msisdn)
			
			assert.Equal(t, tt.expectedLength, len(otp), "OTP length should match expected")
			assert.Equal(t, tt.expectedOTP, otp, "OTP should match expected value")
			
			// Verify OTP contains only digits (if not empty)
			if len(otp) > 0 {
				for _, char := range otp {
					assert.True(t, char >= '0' && char <= '9', "OTP should contain only digits")
				}
			}
		})
	}
}

func TestGenerateDummyOTP_Consistency(t *testing.T) {
	t.Run("Same input produces same output", func(t *testing.T) {
		msisdn := "628111234567"
		
		otp1 := GenerateDummyOTP(msisdn)
		otp2 := GenerateDummyOTP(msisdn)
		
		assert.Equal(t, otp1, otp2, "Same MSISDN should always produce same OTP")
	})

	t.Run("Different inputs produce different outputs", func(t *testing.T) {
		msisdn1 := "628111234567"
		msisdn2 := "628111234568"
		
		otp1 := GenerateDummyOTP(msisdn1)
		otp2 := GenerateDummyOTP(msisdn2)
		
		assert.NotEqual(t, otp1, otp2, "Different MSISDNs should produce different OTPs")
	})
}

func TestPREFIXES_Constant(t *testing.T) {
	t.Run("PREFIXES constant validation", func(t *testing.T) {
		// Test that all expected Telkomsel prefixes are present
		expectedPrefixes := []int{811, 812, 813, 821, 822, 823, 851, 852, 853}
		
		assert.Equal(t, len(expectedPrefixes), len(PREFIXES.TELKOMSEL), "Should have correct number of prefixes")
		
		for _, expected := range expectedPrefixes {
			found := false
			for _, actual := range PREFIXES.TELKOMSEL {
				if actual == expected {
					found = true
					break
				}
			}
			assert.True(t, found, "Prefix %d should be in TELKOMSEL prefixes", expected)
		}
	})

	t.Run("No duplicate prefixes", func(t *testing.T) {
		prefixMap := make(map[int]bool)
		for _, prefix := range PREFIXES.TELKOMSEL {
			assert.False(t, prefixMap[prefix], "Prefix %d should not be duplicated", prefix)
			prefixMap[prefix] = true
		}
	})

	t.Run("All prefixes are 3 digits", func(t *testing.T) {
		for _, prefix := range PREFIXES.TELKOMSEL {
			assert.GreaterOrEqual(t, prefix, 100, "Prefix %d should be at least 3 digits", prefix)
			assert.LessOrEqual(t, prefix, 999, "Prefix %d should be at most 3 digits", prefix)
		}
	})
}

// Benchmark tests
func BenchmarkValidateMSISDN(b *testing.B) {
	msisdn := "628111234567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateMSISDN(msisdn)
	}
}

func BenchmarkGenerateDummyOTP(b *testing.B) {
	msisdn := "628111234567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateDummyOTP(msisdn)
	}
}