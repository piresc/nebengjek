package service

import (
	"context"
	"testing"
	"time"
)

func TestCalculateFare(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()

	// Test case 1: Normal trip with multiple locations
	trip1 := &Trip{
		ID: "trip1",
		DriverID: "driver1",
		CustomerID: "customer1",
		StartTime: time.Now(),
		Locations: []Location{
			{Latitude: -6.2088, Longitude: 106.8456, Timestamp: time.Now()},
			{Latitude: -6.2000, Longitude: 106.8400, Timestamp: time.Now().Add(time.Minute)},
			{Latitude: -6.1900, Longitude: 106.8350, Timestamp: time.Now().Add(2 * time.Minute)},
		},
	}

	err := service.CalculateFare(ctx, trip1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify base fare calculation (approximately 2.5km * 3000 IDR)
	expectedBaseFare := 7500.0 // Approximate value
	if trip1.BaseFare < expectedBaseFare*0.9 || trip1.BaseFare > expectedBaseFare*1.1 {
		t.Errorf("Expected base fare around %v, got %v", expectedBaseFare, trip1.BaseFare)
	}

	// Verify admin fee calculation (5%)
	expectedAdminFee := trip1.AdjustedFare * AdminFeePercentage
	if trip1.AdminFee != expectedAdminFee {
		t.Errorf("Expected admin fee %v, got %v", expectedAdminFee, trip1.AdminFee)
	}

	// Test case 2: Trip with insufficient locations
	trip2 := &Trip{
		ID: "trip2",
		Locations: []Location{
			{Latitude: -6.2088, Longitude: 106.8456, Timestamp: time.Now()},
		},
	}

	err = service.CalculateFare(ctx, trip2)
	if err == nil {
		t.Error("Expected error for insufficient locations, got nil")
	}
}

func TestAdjustFare(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()

	trip := &Trip{
		ID: "trip1",
		BaseFare: 10000, // 10000 IDR
		AdjustedFare: 10000,
	}

	// Test case 1: Valid adjustment (80%)
	err := service.AdjustFare(ctx, trip, 0.8)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedAdjustedFare := 8000.0 // 80% of 10000
	if trip.AdjustedFare != expectedAdjustedFare {
		t.Errorf("Expected adjusted fare %v, got %v", expectedAdjustedFare, trip.AdjustedFare)
	}

	expectedAdminFee := expectedAdjustedFare * AdminFeePercentage
	if trip.AdminFee != expectedAdminFee {
		t.Errorf("Expected admin fee %v, got %v", expectedAdminFee, trip.AdminFee)
	}

	// Test case 2: Invalid adjustment (>100%)
	err = service.AdjustFare(ctx, trip, 1.2)
	if err == nil {
		t.Error("Expected error for adjustment > 100%, got nil")
	}

	// Test case 3: Invalid adjustment (0%)
	err = service.AdjustFare(ctx, trip, 0)
	if err == nil {
		t.Error("Expected error for 0% adjustment, got nil")
	}
}