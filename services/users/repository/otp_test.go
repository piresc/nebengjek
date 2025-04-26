package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// setupMiniredis creates a new miniredis server and returns a Redis client connected to it
func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func setupOTPRepoTest(t *testing.T) (*UserRepo, *miniredis.Miniredis) {
	// Create miniredis server
	mr, client := setupMiniredis(t)

	// Create Redis client wrapper
	redisClient := &database.RedisClient{
		Client: client,
	}

	// Create repo with Redis client
	repo := &UserRepo{
		redisClient: redisClient,
	}

	return repo, mr
}

func TestCreateOTP(t *testing.T) {
	// Setup
	repo, mr := setupOTPRepoTest(t)
	defer mr.Close()

	// Test data
	otp := models.OTP{
		MSISDN: "+628123456789",
		Code:   "123456",
	}

	// Execute
	err := repo.CreateOTP(context.Background(), &otp)
	
	// Assert
	assert.NoError(t, err)
	
	// Verify data was stored in Redis
	key := fmt.Sprintf(constants.KeyUserOTP, otp.MSISDN)
	val, err := mr.Get(key)
	assert.NoError(t, err)
	
	var storedOTP models.OTP
	err = json.Unmarshal([]byte(val), &storedOTP)
	assert.NoError(t, err)
	assert.Equal(t, otp.MSISDN, storedOTP.MSISDN)
	assert.Equal(t, otp.Code, storedOTP.Code)
	
	// Verify TTL
	ttl := mr.TTL(key)
	assert.True(t, ttl > 0)
}

func TestCreateOTP_RedisError(t *testing.T) {
	// Setup
	repo, mr := setupOTPRepoTest(t)
	
	// Force Redis to fail by closing the connection
	mr.Close()

	// Test data
	otp := models.OTP{
		MSISDN: "+628123456789",
		Code:   "123456",
	}

	// Execute
	err := repo.CreateOTP(context.Background(), &otp)
	
	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store OTP in Redis")
}

func TestGetOTP(t *testing.T) {
	testCases := []struct {
		name      string
		msisdn    string
		code      string
		setupFunc func(mr *miniredis.Miniredis)
		wantErr   bool
		wantOTP   *models.OTP
	}{
		{
			name:   "Success",
			msisdn: "+628123456789",
			code:   "123456",
			setupFunc: func(mr *miniredis.Miniredis) {
				otp := models.OTP{
					MSISDN: "+628123456789",
					Code:   "123456",
				}
				otpJSON, _ := json.Marshal(otp)
				key := fmt.Sprintf(constants.KeyUserOTP, otp.MSISDN)
				mr.Set(key, string(otpJSON))
				mr.SetTTL(key, 5*time.Minute)
			},
			wantErr: false,
			wantOTP: &models.OTP{
				MSISDN: "+628123456789",
				Code:   "123456",
			},
		},
		{
			name:   "OTP Not Found",
			msisdn: "+628123456790",
			code:   "123456",
			setupFunc: func(mr *miniredis.Miniredis) {
				// No setup - OTP doesn't exist
			},
			wantErr: true,
			wantOTP: nil,
		},
		{
			name:   "Invalid JSON",
			msisdn: "+628123456791",
			code:   "123456",
			setupFunc: func(mr *miniredis.Miniredis) {
				key := fmt.Sprintf(constants.KeyUserOTP, "+628123456791")
				mr.Set(key, "invalid json")
			},
			wantErr: true,
			wantOTP: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			repo, mr := setupOTPRepoTest(t)
			defer mr.Close()
			
			// Setup test case
			tc.setupFunc(mr)

			// Execute
			otp, err := repo.GetOTP(context.Background(), tc.msisdn, tc.code)

			// Assert
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, otp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, otp)
				assert.Equal(t, tc.wantOTP.MSISDN, otp.MSISDN)
				assert.Equal(t, tc.wantOTP.Code, otp.Code)
			}
		})
	}
}

func TestMarkOTPVerified(t *testing.T) {
	testCases := []struct {
		name      string
		msisdn    string
		code      string
		setupFunc func(mr *miniredis.Miniredis)
		wantErr   bool
	}{
		{
			name:   "Success",
			msisdn: "+628123456789",
			code:   "123456",
			setupFunc: func(mr *miniredis.Miniredis) {
				otp := models.OTP{
					MSISDN: "+628123456789",
					Code:   "123456",
				}
				otpJSON, _ := json.Marshal(otp)
				key := fmt.Sprintf(constants.KeyUserOTP, otp.MSISDN)
				mr.Set(key, string(otpJSON))
			},
			wantErr: false,
		},
		{
			name:   "Redis Error",
			msisdn: "+628123456790",
			code:   "123456",
			setupFunc: func(mr *miniredis.Miniredis) {
				// Will be closed in the test
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			repo, mr := setupOTPRepoTest(t)
			
			// Setup test case
			tc.setupFunc(mr)
			
			// For the Redis error test, close the connection after setup
			if tc.name == "Redis Error" {
				mr.Close()
			} else {
				defer mr.Close()
			}

			// Execute
			err := repo.MarkOTPVerified(context.Background(), tc.msisdn, tc.code)

			// Assert
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to delete OTP")
			} else {
				assert.NoError(t, err)
				
				// Verify OTP is deleted from Redis
				key := fmt.Sprintf(constants.KeyUserOTP, tc.msisdn)
				assert.False(t, mr.Exists(key))
			}
		})
	}
}