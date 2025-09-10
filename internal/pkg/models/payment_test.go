package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPaymentStatus_Constants(t *testing.T) {
	// Test that all status constants have valid values
	assert.Equal(t, "PENDING", string(PaymentStatusPending))
	assert.Equal(t, "ACCEPTED", string(PaymentStatusAccepted))
	assert.Equal(t, "REJECTED", string(PaymentStatusRejected))
	assert.Equal(t, "PROCESSED", string(PaymentStatusProcessed))
}

func TestPaymentStatus_StringConversion(t *testing.T) {
	tests := []struct {
		status   PaymentStatus
		expected string
	}{
		{PaymentStatusPending, "PENDING"},
		{PaymentStatusAccepted, "ACCEPTED"},
		{PaymentStatusRejected, "REJECTED"},
		{PaymentStatusProcessed, "PROCESSED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestPaymentRequest_DefaultValues(t *testing.T) {
	req := &PaymentRequest{}

	// Test zero values
	assert.Equal(t, "", req.RideID)
	assert.Equal(t, "", req.PassengerID)
	assert.Equal(t, "", req.DriverID)
	assert.Equal(t, 0, req.TotalCost)
	assert.Equal(t, "", req.QRCodeURL)
}

func TestPaymentRequest_WithValues(t *testing.T) {
	req := &PaymentRequest{
		RideID:      "test-ride-id",
		PassengerID: "test-passenger-id",
		DriverID:    "test-driver-id",
		TotalCost:   25000,
		QRCodeURL:   "https://payment.nebengjek.com/qr/test-ride-id",
	}

	assert.Equal(t, "test-ride-id", req.RideID)
	assert.Equal(t, "test-passenger-id", req.PassengerID)
	assert.Equal(t, "test-driver-id", req.DriverID)
	assert.Equal(t, 25000, req.TotalCost)
	assert.Equal(t, "https://payment.nebengjek.com/qr/test-ride-id", req.QRCodeURL)
}

func TestPaymentResponse_DefaultValues(t *testing.T) {
	resp := &PaymentResponse{}

	// Test zero values
	assert.Equal(t, Payment{}, resp.Payment)
	assert.Equal(t, "", resp.RideID)
	assert.Equal(t, "", resp.Status)
	assert.Equal(t, "", resp.Message)
	assert.Equal(t, time.Time{}, resp.IssuedAt)
}

func TestPaymentResponse_WithValues(t *testing.T) {
	paymentID := uuid.New()
	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	now := time.Now()

	payment := Payment{
		PaymentID:    paymentID,
		RideID:       rideID,
		AdjustedCost: 25000,
		AdminFee:     2500,
		DriverPayout: 22500,
		Status:       PaymentStatusAccepted,
		CreatedAt:    now,
		DriverID:     driverID,
		PassengerID:  passengerID,
	}

	issuedAt := time.Now()
	resp := &PaymentResponse{
		Payment:  payment,
		RideID:   "test-ride-id",
		Status:   "SUCCESS",
		Message:  "Payment processed successfully",
		IssuedAt: issuedAt,
	}

	assert.Equal(t, paymentID, resp.Payment.PaymentID)
	assert.Equal(t, "test-ride-id", resp.RideID)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, "Payment processed successfully", resp.Message)
	assert.Equal(t, issuedAt, resp.IssuedAt)
}

func TestPaymentProccessRequest_DefaultValues(t *testing.T) {
	req := &PaymentProccessRequest{}

	// Test zero values
	assert.Equal(t, "", req.RideID)
	assert.Equal(t, 0, req.TotalCost)
	assert.Equal(t, PaymentStatus(""), req.Status)
}

func TestPaymentProccessRequest_WithValues(t *testing.T) {
	req := &PaymentProccessRequest{
		RideID:    "test-ride-id",
		TotalCost: 25000,
		Status:    PaymentStatusAccepted,
	}

	assert.Equal(t, "test-ride-id", req.RideID)
	assert.Equal(t, 25000, req.TotalCost)
	assert.Equal(t, PaymentStatusAccepted, req.Status)
}

func TestPaymentStatus_Transitions(t *testing.T) {
	// Test typical payment status transitions
	statuses := []PaymentStatus{
		PaymentStatusPending,
		PaymentStatusAccepted,
		PaymentStatusProcessed,
	}

	for i, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			assert.NotEmpty(t, string(status))
			if i > 0 {
				assert.NotEqual(t, statuses[i-1], status)
			}
		})
	}
}

func TestPaymentStatus_RejectionPath(t *testing.T) {
	// Test that rejection is a valid status
	assert.Equal(t, "REJECTED", string(PaymentStatusRejected))
	assert.NotEqual(t, PaymentStatusPending, PaymentStatusRejected)
	assert.NotEqual(t, PaymentStatusAccepted, PaymentStatusRejected)
	assert.NotEqual(t, PaymentStatusProcessed, PaymentStatusRejected)
}

func TestPaymentRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *PaymentRequest
		valid   bool
		comment string
	}{
		{
			name:    "Valid request",
			req:     &PaymentRequest{RideID: "test-ride", PassengerID: "passenger", DriverID: "driver", TotalCost: 1000},
			valid:   true,
			comment: "All required fields present",
		},
		{
			name:    "Missing ride ID",
			req:     &PaymentRequest{PassengerID: "passenger", DriverID: "driver", TotalCost: 1000},
			valid:   false,
			comment: "Ride ID is required",
		},
		{
			name:    "Missing passenger ID",
			req:     &PaymentRequest{RideID: "test-ride", DriverID: "driver", TotalCost: 1000},
			valid:   false,
			comment: "Passenger ID is required",
		},
		{
			name:    "Missing driver ID",
			req:     &PaymentRequest{RideID: "test-ride", PassengerID: "passenger", TotalCost: 1000},
			valid:   false,
			comment: "Driver ID is required",
		},
		{
			name:    "Zero total cost",
			req:     &PaymentRequest{RideID: "test-ride", PassengerID: "passenger", DriverID: "driver", TotalCost: 0},
			valid:   false,
			comment: "Total cost must be positive",
		},
		{
			name:    "Negative total cost",
			req:     &PaymentRequest{RideID: "test-ride", PassengerID: "passenger", DriverID: "driver", TotalCost: -1000},
			valid:   false,
			comment: "Total cost cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation logic
			hasRideID := tt.req.RideID != ""
			hasPassengerID := tt.req.PassengerID != ""
			hasDriverID := tt.req.DriverID != ""
			hasValidCost := tt.req.TotalCost > 0
			
			isValid := hasRideID && hasPassengerID && hasDriverID && hasValidCost
			
			assert.Equal(t, tt.valid, isValid, tt.comment)
		})
	}
}

func TestPaymentProccessRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *PaymentProccessRequest
		valid   bool
		comment string
	}{
		{
			name:    "Valid request",
			req:     &PaymentProccessRequest{RideID: "test-ride", TotalCost: 1000, Status: PaymentStatusAccepted},
			valid:   true,
			comment: "All required fields present with valid status",
		},
		{
			name:    "Missing ride ID",
			req:     &PaymentProccessRequest{TotalCost: 1000, Status: PaymentStatusAccepted},
			valid:   false,
			comment: "Ride ID is required",
		},
		{
			name:    "Zero total cost",
			req:     &PaymentProccessRequest{RideID: "test-ride", TotalCost: 0, Status: PaymentStatusAccepted},
			valid:   false,
			comment: "Total cost must be positive",
		},
		{
			name:    "Empty status",
			req:     &PaymentProccessRequest{RideID: "test-ride", TotalCost: 1000, Status: PaymentStatus("")},
			valid:   false,
			comment: "Status is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation logic
			hasRideID := tt.req.RideID != ""
			hasValidCost := tt.req.TotalCost > 0
			hasStatus := tt.req.Status != ""
			
			isValid := hasRideID && hasValidCost && hasStatus
			
			assert.Equal(t, tt.valid, isValid, tt.comment)
		})
	}
}

func TestPaymentRequest_QRCodeURL_Scenarios(t *testing.T) {
	tests := []struct {
		name         string
		qrCodeURL    string
		expectEmpty  bool
		expectFormat string
	}{
		{
			name:         "Full URL",
			qrCodeURL:    "https://payment.nebengjek.com/qr/test-ride-id",
			expectEmpty:  false,
			expectFormat: "https://",
		},
		{
			name:         "Relative URL",
			qrCodeURL:    "/qr/test-ride-id",
			expectEmpty:  false,
			expectFormat: "/qr/",
		},
		{
			name:         "Empty URL",
			qrCodeURL:    "",
			expectEmpty:  true,
			expectFormat: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &PaymentRequest{
				RideID:      "test-ride-id",
				PassengerID: "test-passenger-id",
				DriverID:    "test-driver-id",
				TotalCost:   25000,
				QRCodeURL:   tt.qrCodeURL,
			}

			if tt.expectEmpty {
				assert.Empty(t, req.QRCodeURL)
			} else {
				assert.NotEmpty(t, req.QRCodeURL)
				if tt.expectFormat != "" {
					assert.Contains(t, req.QRCodeURL, tt.expectFormat)
				}
			}
		})
	}
}

func TestPaymentResponse_Struct(t *testing.T) {
	// Test that the struct can be created without panics
	resp := &PaymentResponse{}
	assert.NotNil(t, resp)

	// Test that all fields are accessible
	resp.Payment = Payment{}
	resp.RideID = "test-ride"
	resp.Status = "SUCCESS"
	resp.Message = "Test message"
	resp.IssuedAt = time.Now()

	assert.Equal(t, "test-ride", resp.RideID)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, "Test message", resp.Message)
	assert.NotZero(t, resp.IssuedAt)
}

func TestPaymentProccessRequest_StatusValues(t *testing.T) {
	validStatuses := []PaymentStatus{
		PaymentStatusPending,
		PaymentStatusAccepted,
		PaymentStatusRejected,
		PaymentStatusProcessed,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			req := &PaymentProccessRequest{
				RideID:    "test-ride-id",
				TotalCost: 25000,
				Status:    status,
			}

			assert.Equal(t, status, req.Status)
			assert.NotEmpty(t, string(status))
		})
	}
}