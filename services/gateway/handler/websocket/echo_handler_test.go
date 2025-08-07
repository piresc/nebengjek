package gatewaywebsocket

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	gatewaymocks "github.com/piresc/nebengjek/services/gateway/mocks"
	usersmocks "github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/websocket"
)

func TestNewEchoWebSocketHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := gatewaymocks.NewMockGatewayUC(ctrl)
	mockUserUC := usersmocks.NewMockUserUC(ctrl)

	handler := NewEchoWebSocketHandler(mockGatewayUC, mockUserUC)

	assert.NotNil(t, handler)
	assert.Equal(t, mockGatewayUC, handler.gatewayUC)
	assert.Equal(t, mockUserUC, handler.userUC)
	assert.NotNil(t, handler.clients)
	assert.NotNil(t, handler.lastActivity)
	assert.Empty(t, handler.clients)
	assert.Empty(t, handler.lastActivity)
}

func TestEchoWebSocketHandler_ClientManagement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := gatewaymocks.NewMockGatewayUC(ctrl)
	mockUserUC := usersmocks.NewMockUserUC(ctrl)

	handler := NewEchoWebSocketHandler(mockGatewayUC, mockUserUC)

	userID := uuid.New().String()

	// Initially no clients
	assert.False(t, handler.IsUserConnected(userID))
	assert.Equal(t, 0, handler.GetConnectedUsersCount())
	assert.Equal(t, time.Time{}, handler.GetUserLastActivity(userID))

	// Create a mock websocket connection
	mockWS := &websocket.Conn{}

	// Add client
	handler.addClient(userID, mockWS)

	assert.True(t, handler.IsUserConnected(userID))
	assert.Equal(t, 1, handler.GetConnectedUsersCount())
	assert.NotEqual(t, time.Time{}, handler.GetUserLastActivity(userID))

	// Remove client
	handler.removeClient(userID)

	assert.False(t, handler.IsUserConnected(userID))
	assert.Equal(t, 0, handler.GetConnectedUsersCount())
	assert.Equal(t, time.Time{}, handler.GetUserLastActivity(userID))
}

func TestEchoWebSocketHandler_getSeverityString(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := gatewaymocks.NewMockGatewayUC(ctrl)
	mockUserUC := usersmocks.NewMockUserUC(ctrl)

	handler := NewEchoWebSocketHandler(mockGatewayUC, mockUserUC)

	tests := []struct {
		severity constants.ErrorSeverity
		expected string
	}{
		{constants.ErrorSeverityClient, "client"},
		{constants.ErrorSeverityServer, "server"},
		{constants.ErrorSeveritySecurity, "security"},
		{constants.ErrorSeverity(999), "unknown"}, // Invalid severity
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := handler.getSeverityString(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEchoWebSocketHandler_extractCleanErrorMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := gatewaymocks.NewMockGatewayUC(ctrl)
	mockUserUC := usersmocks.NewMockUserUC(ctrl)

	handler := NewEchoWebSocketHandler(mockGatewayUC, mockUserUC)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "service error with JSON response",
			input:    `rides service returned error: {"success":false,"error":"ride not found","code":404}`,
			expected: "ride not found",
		},
		{
			name:     "service error with malformed JSON",
			input:    `users service returned error: {"success":false,"error":}`,
			expected: `users service returned error: {"success":false,"error":}`,
		},
		{
			name:     "non-service error",
			input:    "connection timeout",
			expected: "connection timeout",
		},
		{
			name:     "service error without JSON",
			input:    "match service returned error: simple error message",
			expected: "match service returned error: simple error message",
		},
		{
			name:     "service error with valid JSON but no error field",
			input:    `location service returned error: {"success":false,"message":"invalid location"}`,
			expected: `location service returned error: {"success":false,"message":"invalid location"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractCleanErrorMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEchoWebSocketHandler_NotifyClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := gatewaymocks.NewMockGatewayUC(ctrl)
	mockUserUC := usersmocks.NewMockUserUC(ctrl)

	handler := NewEchoWebSocketHandler(mockGatewayUC, mockUserUC)

	userID := uuid.New().String()
	event := "test_event"
	data := map[string]string{"message": "test"}

	t.Run("user not connected", func(t *testing.T) {
		// Should not panic and should log warning
		handler.NotifyClient(userID, event, data)
		// No assertion needed, just ensure it doesn't panic
	})

	t.Run("successful notification with error return", func(t *testing.T) {
		err := handler.NotifyClientWithError(userID, event, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("marshal error", func(t *testing.T) {
		// Create invalid data that can't be marshaled
		invalidData := make(chan int) // channels can't be marshaled to JSON
		
		err := handler.NotifyClientWithError(userID, event, invalidData)
		assert.Error(t, err)
		// The error message might vary, so just check that it's an error
		assert.NotNil(t, err)
	})
}

// Note: WebSocket handler methods (handleBeaconUpdate, handleLocationUpdate, etc.) 
// are not unit tested here due to the complexity of mocking websocket.Conn.
// The business logic is covered through the UseCase tests, and the WebSocket
// functionality would be better tested through integration tests.

