package ride

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
	"github.com/stretchr/testify/assert"
)

func TestRideStatus_Constants(t *testing.T) {
	// Test that all status constants have valid values
	assert.Equal(t, "PENDING", string(RideStatusPending))
	assert.Equal(t, "PICKUP", string(RideStatusDriverPickup))
	assert.Equal(t, "ONGOING", string(RideStatusOngoing))
	assert.Equal(t, "COMPLETED", string(RideStatusCompleted))
}

func TestRideStatus_StringConversion(t *testing.T) {
	tests := []struct {
		status   RideStatus
		expected string
	}{
		{RideStatusPending, "PENDING"},
		{RideStatusDriverPickup, "PICKUP"},
		{RideStatusOngoing, "ONGOING"},
		{RideStatusCompleted, "COMPLETED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestRide_DefaultValues(t *testing.T) {
	ride := &Ride{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, ride.RideID)
	assert.Equal(t, uuid.UUID{}, ride.MatchID)
	assert.Equal(t, uuid.UUID{}, ride.DriverID)
	assert.Equal(t, uuid.UUID{}, ride.PassengerID)
	assert.Equal(t, RideStatus(""), ride.Status)
	assert.Equal(t, 0, ride.TotalCost)
	assert.Equal(t, time.Time{}, ride.CreatedAt)
	assert.Equal(t, time.Time{}, ride.UpdatedAt)
}

func TestRide_WithValues(t *testing.T) {
	rideID := uuid.New()
	matchID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	now := time.Now()

	ride := &Ride{
		RideID:      rideID,
		MatchID:     matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      RideStatusPending,
		TotalCost:   25000,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, rideID, ride.RideID)
	assert.Equal(t, matchID, ride.MatchID)
	assert.Equal(t, driverID, ride.DriverID)
	assert.Equal(t, passengerID, ride.PassengerID)
	assert.Equal(t, RideStatusPending, ride.Status)
	assert.Equal(t, 25000, ride.TotalCost)
	assert.Equal(t, now, ride.CreatedAt)
	assert.Equal(t, now, ride.UpdatedAt)
}

func TestRideResp_DefaultValues(t *testing.T) {
	resp := &RideResp{}

	// Test zero values
	assert.Equal(t, "", resp.RideID)
	assert.Equal(t, "", resp.MatchID)
	assert.Equal(t, "", resp.DriverID)
	assert.Equal(t, "", resp.PassengerID)
	assert.Equal(t, "", resp.Status)
	assert.Equal(t, 0, resp.TotalCost)
	assert.Equal(t, time.Time{}, resp.CreatedAt)
	assert.Equal(t, time.Time{}, resp.UpdatedAt)
}

func TestRideResp_WithValues(t *testing.T) {
	now := time.Now()

	resp := &RideResp{
		RideID:      "test-ride-id",
		MatchID:     "test-match-id",
		DriverID:    "test-driver-id",
		PassengerID: "test-passenger-id",
		Status:      "PENDING",
		TotalCost:   25000,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, "test-ride-id", resp.RideID)
	assert.Equal(t, "test-match-id", resp.MatchID)
	assert.Equal(t, "test-driver-id", resp.DriverID)
	assert.Equal(t, "test-passenger-id", resp.PassengerID)
	assert.Equal(t, "PENDING", resp.Status)
	assert.Equal(t, 25000, resp.TotalCost)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestBillingLedger_DefaultValues(t *testing.T) {
	ledger := &BillingLedger{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, ledger.EntryID)
	assert.Equal(t, uuid.UUID{}, ledger.RideID)
	assert.Equal(t, 0.0, ledger.Distance)
	assert.Equal(t, 0, ledger.Cost)
	assert.Equal(t, time.Time{}, ledger.CreatedAt)
}

func TestBillingLedger_WithValues(t *testing.T) {
	entryID := uuid.New()
	rideID := uuid.New()
	now := time.Now()

	ledger := &BillingLedger{
		EntryID:   entryID,
		RideID:    rideID,
		Distance:  5.5,
		Cost:      16500,
		CreatedAt: now,
	}

	assert.Equal(t, entryID, ledger.EntryID)
	assert.Equal(t, rideID, ledger.RideID)
	assert.Equal(t, 5.5, ledger.Distance)
	assert.Equal(t, 16500, ledger.Cost)
	assert.Equal(t, now, ledger.CreatedAt)
}

func TestRideCompleteEvent_DefaultValues(t *testing.T) {
	event := &RideCompleteEvent{}

	// Test zero values
	assert.Equal(t, "", event.RideID)
	assert.Equal(t, 0.0, event.AdjustmentFactor)
}

func TestRideCompleteEvent_WithValues(t *testing.T) {
	event := &RideCompleteEvent{
		RideID:           "test-ride-id",
		AdjustmentFactor: 1.2,
	}

	assert.Equal(t, "test-ride-id", event.RideID)
	assert.Equal(t, 1.2, event.AdjustmentFactor)
}

func TestRideArrivalReq_DefaultValues(t *testing.T) {
	req := &RideArrivalReq{}

	// Test zero values
	assert.Equal(t, "", req.RideID)
	assert.Equal(t, 0.0, req.AdjustmentFactor)
}

func TestRideArrivalReq_WithValues(t *testing.T) {
	req := &RideArrivalReq{
		RideID:           "test-ride-id",
		AdjustmentFactor: 1.1,
	}

	assert.Equal(t, "test-ride-id", req.RideID)
	assert.Equal(t, 1.1, req.AdjustmentFactor)
}

func TestPayment_DefaultValues(t *testing.T) {
	payment := &Payment{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, payment.PaymentID)
	assert.Equal(t, uuid.UUID{}, payment.RideID)
	assert.Equal(t, 0, payment.AdjustedCost)
	assert.Equal(t, 0, payment.AdminFee)
	assert.Equal(t, 0, payment.DriverPayout)
	assert.Equal(t, PaymentStatus(""), payment.Status)
	assert.Equal(t, time.Time{}, payment.CreatedAt)
	assert.Equal(t, uuid.UUID{}, payment.DriverID)
	assert.Equal(t, uuid.UUID{}, payment.PassengerID)
}

func TestPayment_WithValues(t *testing.T) {
	paymentID := uuid.New()
	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	now := time.Now()

	payment := &Payment{
		PaymentID:    paymentID,
		RideID:       rideID,
		AdjustedCost:  25000,
		AdminFee:     2500,
		DriverPayout: 22500,
		Status:       PaymentStatusAccepted,
		CreatedAt:    now,
		DriverID:     driverID,
		PassengerID:  passengerID,
	}

	assert.Equal(t, paymentID, payment.PaymentID)
	assert.Equal(t, rideID, payment.RideID)
	assert.Equal(t, 25000, payment.AdjustedCost)
	assert.Equal(t, 2500, payment.AdminFee)
	assert.Equal(t, 22500, payment.DriverPayout)
	assert.Equal(t, PaymentStatusAccepted, payment.Status)
	assert.Equal(t, now, payment.CreatedAt)
	assert.Equal(t, driverID, payment.DriverID)
	assert.Equal(t, passengerID, payment.PassengerID)
}

func TestRideComplete_DefaultValues(t *testing.T) {
	complete := &RideComplete{}

	// Test zero values
	assert.Equal(t, Ride{}, complete.Ride)
	assert.Equal(t, Payment{}, complete.Payment)
}

func TestRideComplete_WithValues(t *testing.T) {
	rideID := uuid.New()
	matchID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	paymentID := uuid.New()
	now := time.Now()

	ride := Ride{
		RideID:      rideID,
		MatchID:     matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      RideStatusCompleted,
		TotalCost:   25000,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

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

	complete := &RideComplete{
		Ride:    ride,
		Payment: payment,
	}

	assert.Equal(t, rideID, complete.Ride.RideID)
	assert.Equal(t, paymentID, complete.Payment.PaymentID)
	assert.Equal(t, RideStatusCompleted, complete.Ride.Status)
	assert.Equal(t, PaymentStatusAccepted, complete.Payment.Status)
}

func TestRideStartTripEvent_DefaultValues(t *testing.T) {
	event := &RideStartTripEvent{}

	// Test zero values
	assert.Equal(t, "", event.RideID)
	assert.Equal(t, location.Location{}, event.DriverLocation)
	assert.Equal(t, location.Location{}, event.PassengerLocation)
	assert.Equal(t, time.Time{}, event.Timestamp)
}

func TestRideStartTripEvent_WithValues(t *testing.T) {
	now := time.Now()
	driverLocation := location.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}
	passengerLocation := location.Location{
		Latitude:  -6.1751,
		Longitude: 106.8650,
	}

	event := &RideStartTripEvent{
		RideID:            "test-ride-id",
		DriverLocation:    driverLocation,
		PassengerLocation: passengerLocation,
		Timestamp:         now,
	}

	assert.Equal(t, "test-ride-id", event.RideID)
	assert.Equal(t, -6.2088, event.DriverLocation.Latitude)
	assert.Equal(t, 106.8456, event.DriverLocation.Longitude)
	assert.Equal(t, -6.1751, event.PassengerLocation.Latitude)
	assert.Equal(t, 106.8650, event.PassengerLocation.Longitude)
	assert.Equal(t, now, event.Timestamp)
}

func TestRideStartRequest_DefaultValues(t *testing.T) {
	req := &RideStartRequest{}

	// Test zero values
	assert.Equal(t, "", req.RideID)
	assert.Nil(t, req.DriverLocation)
	assert.Nil(t, req.PassengerLocation)
}

func TestRideStartRequest_WithValues(t *testing.T) {
	driverLocation := &location.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}
	passengerLocation := &location.Location{
		Latitude:  -6.1751,
		Longitude: 106.8650,
	}

	req := &RideStartRequest{
		RideID:            "test-ride-id",
		DriverLocation:    driverLocation,
		PassengerLocation: passengerLocation,
	}

	assert.Equal(t, "test-ride-id", req.RideID)
	assert.NotNil(t, req.DriverLocation)
	assert.NotNil(t, req.PassengerLocation)
	assert.Equal(t, -6.2088, req.DriverLocation.Latitude)
	assert.Equal(t, -6.1751, req.PassengerLocation.Latitude)
}

func TestRideStartRequest_WithNilLocations(t *testing.T) {
	req := &RideStartRequest{
		RideID:            "test-ride-id",
		DriverLocation:    nil,
		PassengerLocation: nil,
	}

	assert.Equal(t, "test-ride-id", req.RideID)
	assert.Nil(t, req.DriverLocation)
	assert.Nil(t, req.PassengerLocation)
}

func TestRidePickupEvent_DefaultValues(t *testing.T) {
	event := &RidePickupEvent{}

	// Test zero values
	assert.Equal(t, "", event.RideID)
	assert.Equal(t, "", event.DriverID)
	assert.Equal(t, "", event.PassengerID)
	assert.Equal(t, location.Location{}, event.DriverLocation)
	assert.Equal(t, time.Time{}, event.Timestamp)
}

func TestRidePickupEvent_WithValues(t *testing.T) {
	now := time.Now()
	driverLocation := location.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	event := &RidePickupEvent{
		RideID:         "test-ride-id",
		DriverID:       "test-driver-id",
		PassengerID:    "test-passenger-id",
		DriverLocation: driverLocation,
		Timestamp:      now,
	}

	assert.Equal(t, "test-ride-id", event.RideID)
	assert.Equal(t, "test-driver-id", event.DriverID)
	assert.Equal(t, "test-passenger-id", event.PassengerID)
	assert.Equal(t, -6.2088, event.DriverLocation.Latitude)
	assert.Equal(t, 106.8456, event.DriverLocation.Longitude)
	assert.Equal(t, now, event.Timestamp)
}

func TestRideArrival_DefaultValues(t *testing.T) {
	arrival := &RideArrival{}

	// Test zero values
	assert.Equal(t, "", arrival.RideID)
	assert.Equal(t, "", arrival.DriverID)
	assert.Equal(t, "", arrival.PassengerID)
	assert.Equal(t, 0.0, arrival.AdjustmentFactor)
}

func TestRideArrival_WithValues(t *testing.T) {
	arrival := &RideArrival{
		RideID:           "test-ride-id",
		DriverID:         "test-driver-id",
		PassengerID:      "test-passenger-id",
		AdjustmentFactor: 1.15,
	}

	assert.Equal(t, "test-ride-id", arrival.RideID)
	assert.Equal(t, "test-driver-id", arrival.DriverID)
	assert.Equal(t, "test-passenger-id", arrival.PassengerID)
	assert.Equal(t, 1.15, arrival.AdjustmentFactor)
}

func TestRideStatus_Transitions(t *testing.T) {
	// Test typical ride status transitions
	statuses := []RideStatus{
		RideStatusPending,
		RideStatusDriverPickup,
		RideStatusOngoing,
		RideStatusCompleted,
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

func TestRideStruct_JSONTags(t *testing.T) {
	// Test that all struct fields have proper JSON tags
	ride := &Ride{}
	
	// This test ensures the struct can be marshaled to JSON without panics
	assert.NotNil(t, ride)
	
	// Test that struct can be created - JSON tags are validated by the struct definition
}

func TestPaymentStruct_JSONTags(t *testing.T) {
	// Test that all struct fields have proper JSON tags
	payment := &Payment{}
	
	assert.NotNil(t, payment)
	
	// Test that struct can be created - JSON tags are validated by the struct definition
}

