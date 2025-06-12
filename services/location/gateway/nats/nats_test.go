package nats

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	locationgateway "github.com/piresc/nebengjek/services/location/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNATSConn is a mock implementation of NATSPublisher
type MockNATSConn struct {
	mock.Mock
}

func (m *MockNATSConn) Publish(subject string, data []byte) error {
	args := m.Called(subject, data)
	return args.Error(0)
}

func (m *MockNATSConn) PublishWithOptions(opts natspkg.PublishOptions) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *MockNATSConn) GetConn() *nats.Conn {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*nats.Conn)
}

func (m *MockNATSConn) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	args := m.Called(subject, handler)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nats.Subscription), args.Error(1)
}

func (m *MockNATSConn) Close() {
	m.Called()
}

func TestNewLocationGW(t *testing.T) {
	mockConn := &MockNATSConn{}
	gw := locationgateway.NewLocationGW(mockConn)

	assert.NotNil(t, gw)
	// Just verify the gateway was created successfully
	// We can't access internal fields due to package boundaries
}

func TestLocationGW_PublishLocationAggregate(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		locationData  *models.LocationAggregate
		mockSetup     func(*MockNATSConn)
		expectedError bool
	}{
		{
			name: "Success",
			ctx:  context.Background(),
			locationData: &models.LocationAggregate{
				RideID:    "ride-123",
				Distance:  10.5,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			mockSetup: func(mockConn *MockNATSConn) {
				mockConn.On("PublishWithOptions", mock.MatchedBy(func(opts natspkg.PublishOptions) bool {
					return opts.Subject == "location.aggregate"
				})).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "NATS publish error",
			ctx:  context.Background(),
			locationData: &models.LocationAggregate{
				RideID:    "ride-123",
				Distance:  10.5,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			mockSetup: func(mockConn *MockNATSConn) {
				mockConn.On("PublishWithOptions", mock.MatchedBy(func(opts natspkg.PublishOptions) bool {
					return opts.Subject == "location.aggregate"
				})).Return(errors.New("NATS connection error"))
			},
			expectedError: true,
		},
		{
			name:         "Nil location data",
			ctx:          context.Background(),
			locationData: nil,
			mockSetup: func(mockConn *MockNATSConn) {
				// For nil data, we pass empty struct, so mock should expect it
				mockConn.On("PublishWithOptions", mock.MatchedBy(func(opts natspkg.PublishOptions) bool {
					return opts.Subject == "location.aggregate"
				})).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Context cancelled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			locationData: &models.LocationAggregate{
				RideID:    "ride-123",
				Distance:  10.5,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			mockSetup: func(mockConn *MockNATSConn) {
				// Context cancellation might still call PublishWithOptions, so mock it
				mockConn.On("PublishWithOptions", mock.MatchedBy(func(opts natspkg.PublishOptions) bool {
					return opts.Subject == "location.aggregate"
				})).Return(errors.New("context deadline exceeded"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConn := &MockNATSConn{}
			tt.mockSetup(mockConn)

			gw := locationgateway.NewLocationGW(mockConn)

			var err error
			if tt.locationData != nil {
				err = gw.PublishLocationAggregate(tt.ctx, *tt.locationData)
			} else {
				// For nil data test, pass a zero value
				err = gw.PublishLocationAggregate(tt.ctx, models.LocationAggregate{})
			}

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockConn.AssertExpectations(t)
		})
	}
}

// Note: PublishDriverLocationUpdate and PublishPassengerLocationUpdate methods
// are not part of the LocationGW interface, so these tests are removed.
// Only PublishLocationAggregate is available in the interface.

// Removed TestLocationGW_PublishDriverLocationUpdate - method not in interface
// Removed TestLocationGW_PublishPassengerLocationUpdate - method not in interface

func TestLocationGW_JSONMarshaling(t *testing.T) {
	// Test that the gateway properly marshals data to JSON
	mockConn := &MockNATSConn{}
	gw := locationgateway.NewLocationGW(mockConn)

	locationData := &models.LocationAggregate{
		RideID:    "ride-123",
		Distance:  10.5,
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	// Capture the published data
	var publishedData []byte
	mockConn.On("PublishWithOptions", mock.AnythingOfType("nats.PublishOptions")).Run(func(args mock.Arguments) {
		opts := args.Get(0).(natspkg.PublishOptions)
		publishedData = opts.Data
	}).Return(nil)

	err := gw.PublishLocationAggregate(context.Background(), *locationData)
	assert.NoError(t, err)

	// Verify that the published data is valid JSON
	var unmarshaled models.LocationAggregate
	err = json.Unmarshal(publishedData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, locationData.RideID, unmarshaled.RideID)
	assert.Equal(t, locationData.Distance, unmarshaled.Distance)
	assert.Equal(t, locationData.Latitude, unmarshaled.Latitude)
	assert.Equal(t, locationData.Longitude, unmarshaled.Longitude)

	mockConn.AssertExpectations(t)
}

func BenchmarkLocationGW_PublishLocationAggregate(b *testing.B) {
	mockConn := &MockNATSConn{}
	mockConn.On("PublishWithOptions", mock.AnythingOfType("nats.PublishOptions")).Return(nil)

	gw := locationgateway.NewLocationGW(mockConn)
	locationData := &models.LocationAggregate{
		RideID:    "ride-123",
		Distance:  10.5,
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gw.PublishLocationAggregate(context.Background(), *locationData)
	}
}