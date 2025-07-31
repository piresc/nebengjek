package nats

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	gatewaywebsocket "github.com/piresc/nebengjek/services/gateway/handler/websocket"
	"github.com/stretchr/testify/assert"
)

// MockWebSocketHandler implements the WebSocket handler interface for testing
type MockWebSocketHandler struct {
	notifyClientError error
	notificationsSent []NotificationCall
}

type NotificationCall struct {
	UserID string
	Event  string
	Data   interface{}
}

func (m *MockWebSocketHandler) NotifyClientWithError(userID string, event string, data interface{}) error {
	m.notificationsSent = append(m.notificationsSent, NotificationCall{
		UserID: userID,
		Event:  event,
		Data:   data,
	})
	return m.notifyClientError
}

func TestNewGatewayNotificationHandler(t *testing.T) {
	natsClient := &natspkg.Client{}
	wsHandler := &gatewaywebsocket.EchoWebSocketHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := NewGatewayNotificationHandler(natsClient, wsHandler, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, natsClient, handler.natsClient)
	assert.Equal(t, wsHandler, handler.wsHandler)
	assert.Equal(t, logger, handler.logger)
}

func TestGatewayNotificationHandler_mapNotificationTypeToWSEvent(t *testing.T) {
	handler := &GatewayNotificationHandler{
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	tests := []struct {
		notificationType string
		expectedEvent    string
	}{
		{models.NotificationTypeMatchProposal, "match_proposal"},
		{models.NotificationTypeMatchAccepted, "match_accepted"},
		{models.NotificationTypeMatchRejected, "match_rejected"},
		{models.NotificationTypeRideStarted, "ride_started"},
		{models.NotificationTypeRidePickupArrived, "ride_pickup_arrived"},
		{models.NotificationTypeRideCompleted, "ride_completed"},
		{models.NotificationTypeRideCancelled, "ride_cancelled"},
		{models.NotificationTypePaymentProcessed, "payment_processed"},
		{"unknown_type", "notification"},
	}

	for _, tt := range tests {
		t.Run(tt.notificationType, func(t *testing.T) {
			result := handler.mapNotificationTypeToWSEvent(tt.notificationType)
			assert.Equal(t, tt.expectedEvent, result)
		})
	}
}

func TestGatewayNotificationHandler_handleNotificationDelivery(t *testing.T) {
	tests := []struct {
		name              string
		msgData           []byte
		mockWSError       error
		expectedError     bool
		expectedEventName string
	}{
		{
			name: "successful notification delivery",
			msgData: func() []byte {
				event := models.NotificationDeliveryEvent{
					DeliveryID: uuid.New(),
					UserID:     uuid.New(),
					Type:       models.NotificationTypeMatchProposal,
					Data:       json.RawMessage(`{"match_id": "123"}`),
				}
				data, _ := json.Marshal(event)
				return data
			}(),
			expectedError:     false,
			expectedEventName: "match_proposal",
		},
		{
			name:          "invalid JSON",
			msgData:       []byte("invalid json"),
			expectedError: false, // Should return nil for permanent failures
		},
		{
			name: "websocket delivery failure",
			msgData: func() []byte {
				event := models.NotificationDeliveryEvent{
					DeliveryID: uuid.New(),
					UserID:     uuid.New(),
					Type:       models.NotificationTypeRideStarted,
					Data:       json.RawMessage(`{"ride_id": "456"}`),
				}
				data, _ := json.Marshal(event)
				return data
			}(),
			mockWSError:       errors.New("websocket send failed"),
			expectedError:     true,
			expectedEventName: "ride_started",
		},
		{
			name: "client not connected",
			msgData: func() []byte {
				event := models.NotificationDeliveryEvent{
					DeliveryID: uuid.New(),
					UserID:     uuid.New(),
					Type:       models.NotificationTypePaymentProcessed,
					Data:       json.RawMessage(`{"payment_id": "789"}`),
				}
				data, _ := json.Marshal(event)
				return data
			}(),
			mockWSError:       errors.New("client user-123 not connected"),
			expectedError:     false, // Should acknowledge message when client not connected
			expectedEventName: "payment_processed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWSHandler := &MockWebSocketHandler{
				notifyClientError: tt.mockWSError,
			}

			handler := &GatewayNotificationHandler{
				wsHandler: mockWSHandler,
				logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
			}

			err := handler.handleNotificationDelivery(context.Background(), tt.msgData)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify WebSocket call was made for valid JSON
			if !tt.expectedError && tt.expectedEventName != "" && tt.mockWSError == nil {
				assert.Len(t, mockWSHandler.notificationsSent, 1)
				assert.Equal(t, tt.expectedEventName, mockWSHandler.notificationsSent[0].Event)
			}
		})
	}
}

func TestGatewayNotificationHandler_handleNotificationDelivery_EdgeCases(t *testing.T) {
	t.Run("empty message data", func(t *testing.T) {
		mockWSHandler := &MockWebSocketHandler{}
		handler := &GatewayNotificationHandler{
			wsHandler: mockWSHandler,
			logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
		}

		err := handler.handleNotificationDelivery(context.Background(), []byte{})

		// Should return nil for permanent failures (invalid JSON)
		assert.NoError(t, err)
		assert.Empty(t, mockWSHandler.notificationsSent)
	})

	t.Run("valid JSON but missing required fields", func(t *testing.T) {
		mockWSHandler := &MockWebSocketHandler{}
		handler := &GatewayNotificationHandler{
			wsHandler: mockWSHandler,
			logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
		}

		// Create event with missing UserID
		incompleteEvent := map[string]interface{}{
			"delivery_id": uuid.New().String(),
			"type":        models.NotificationTypeMatchProposal,
			"data":        json.RawMessage(`{"test": "data"}`),
		}
		msgData, _ := json.Marshal(incompleteEvent)

		err := handler.handleNotificationDelivery(context.Background(), msgData)

		// Should handle gracefully - might panic on UserID.String() but test will catch it
		// In real implementation, this would be handled by validation
		assert.NoError(t, err) // Expecting it to complete without major errors
	})

	t.Run("websocket marshaling error", func(t *testing.T) {
		// This is harder to test directly since we're passing json.RawMessage
		// which should always be marshalable. But we can test the error path
		// by using a mock that returns a marshaling-like error.
		mockWSHandler := &MockWebSocketHandler{
			notifyClientError: errors.New("failed to marshal notification data: json error"),
		}

		handler := &GatewayNotificationHandler{
			wsHandler: mockWSHandler,
			logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
		}

		event := models.NotificationDeliveryEvent{
			DeliveryID: uuid.New(),
			UserID:     uuid.New(),
			Type:       models.NotificationTypeMatchProposal,
			Data:       json.RawMessage(`{"match_id": "123"}`),
		}
		msgData, _ := json.Marshal(event)

		err := handler.handleNotificationDelivery(context.Background(), msgData)

		// Should return error for retry since it's not a "not connected" error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deliver notification via WebSocket")
	})
}

func TestGatewayNotificationHandler_Integration(t *testing.T) {
	// Test the complete flow from message data to websocket notification
	t.Run("complete notification flow", func(t *testing.T) {
		mockWSHandler := &MockWebSocketHandler{}
		handler := &GatewayNotificationHandler{
			wsHandler: mockWSHandler,
			logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
		}

		userID := uuid.New()
		deliveryID := uuid.New()
		matchData := map[string]interface{}{
			"match_id":     "match-123",
			"driver_id":    "driver-456",
			"passenger_id": "passenger-789",
		}
		rawData, _ := json.Marshal(matchData)

		event := models.NotificationDeliveryEvent{
			DeliveryID: deliveryID,
			UserID:     userID,
			Type:       models.NotificationTypeMatchAccepted,
			Data:       json.RawMessage(rawData),
		}
		msgData, _ := json.Marshal(event)

		err := handler.handleNotificationDelivery(context.Background(), msgData)

		assert.NoError(t, err)
		assert.Len(t, mockWSHandler.notificationsSent, 1)

		notification := mockWSHandler.notificationsSent[0]
		assert.Equal(t, userID.String(), notification.UserID)
		assert.Equal(t, "match_accepted", notification.Event)
		// The data is passed as json.RawMessage from the event, so check that we have data
		assert.NotNil(t, notification.Data)
	})
}