package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)



func TestNewEchoWebSocketHandler(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)

	// Act
	handler := NewEchoWebSocketHandler(mockUserUC)

	// Assert
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.clients)
	assert.Equal(t, mockUserUC, handler.userUC)
}

func TestEchoWebSocketHandler_HandleWebSocket_MissingUserID(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set user_id in context
	c.Set("role", "driver")

	// Act
	err := handler.HandleWebSocket(c)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Missing user credentials")
}

func TestEchoWebSocketHandler_HandleWebSocket_MissingRole(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set role in context
	c.Set("user_id", uuid.New().String())

	// Act
	err := handler.HandleWebSocket(c)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Missing user credentials")
}

func TestEchoWebSocketHandler_HandleWebSocket_InvalidUserID(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set invalid user_id
	c.Set("user_id", "")
	c.Set("role", "driver")

	// Act
	err := handler.HandleWebSocket(c)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid user credentials")
}

func TestEchoWebSocketHandler_AddAndRemoveClient(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	userID := uuid.New().String()

	// Create a mock websocket connection
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		// Simple echo server for testing
		var msg string
		websocket.Message.Receive(ws, &msg)
		websocket.Message.Send(ws, "echo: "+msg)
	}))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	// Act - Add client
	handler.addClient(userID, ws)

	// Assert - Client added
	handler.mu.RLock()
	_, exists := handler.clients[userID]
	handler.mu.RUnlock()
	assert.True(t, exists)

	// Act - Remove client
	handler.removeClient(userID)

	// Assert - Client removed
	handler.mu.RLock()
	_, exists = handler.clients[userID]
	handler.mu.RUnlock()
	assert.False(t, exists)
}

func TestEchoWebSocketHandler_NotifyClient_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	userID := "123"
	eventType := constants.EventMatchConfirm
	data := map[string]interface{}{
		"match_id": "456",
		"status":   "confirmed",
	}

	// Act & Assert - Should not panic when client doesn't exist
	assert.NotPanics(t, func() {
		handler.NotifyClient(userID, eventType, data)
	})

	// Test that the method handles invalid data gracefully
	assert.NotPanics(t, func() {
		handler.NotifyClient(userID, eventType, make(chan int)) // unmarshalable data
	})
}

func TestEchoWebSocketHandler_NotifyClient_ClientNotFound(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	userID := "nonexistent"
	eventType := constants.EventMatchConfirm
	data := map[string]interface{}{
		"match_id": "456",
		"status":   "confirmed",
	}

	// Act & Assert - Should not panic when client doesn't exist
	assert.NotPanics(t, func() {
		handler.NotifyClient(userID, eventType, data)
	})
}

func TestEchoWebSocketHandler_HandleMessage_LocationUpdate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	userID := uuid.New().String()
	role := "driver"

	// Create a mock websocket connection
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	data := map[string]interface{}{
		"latitude":  -6.2088,
		"longitude": 106.8456,
		"is_active": true,
	}
	dataBytes, _ := json.Marshal(data)
	msg := &models.WSMessage{
		Event: constants.EventLocationUpdate,
		Data: json.RawMessage(dataBytes),
	}

	// Set up mock expectations
	mockUserUC.EXPECT().
		UpdateUserLocation(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = handler.handleMessage(userID, role, ws, msg)

	// Assert
	assert.NoError(t, err)
}

func TestEchoWebSocketHandler_HandleMessage_FinderStatusUpdate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	userID := uuid.New().String()
	role := "passenger"

	// Create a mock websocket connection
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	data := map[string]interface{}{
		"latitude":   -6.2088,
		"longitude":  106.8456,
		"is_finding": true,
	}
	dataBytes, _ := json.Marshal(data)
	msg := &models.WSMessage{
		Event: constants.EventFinderUpdate,
		Data: json.RawMessage(dataBytes),
	}

	// Set up mock expectations
	mockUserUC.EXPECT().
		UpdateFinderStatus(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = handler.handleMessage(userID, role, ws, msg)

	// Assert
	assert.NoError(t, err)
}

func TestEchoWebSocketHandler_HandleMessage_UnknownEvent(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	handler := NewEchoWebSocketHandler(mockUserUC)

	userID := uuid.New().String()
	role := "driver"

	// Create a mock websocket connection
	server := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, err := websocket.Dial(wsURL, "", "http://localhost/")
	require.NoError(t, err)
	defer ws.Close()

	dataBytes, _ := json.Marshal(map[string]interface{}{})
	msg := &models.WSMessage{
		Event: "unknown_event",
		Data:  json.RawMessage(dataBytes),
	}

	// No mock expectations needed for unknown event handling

	// Act
	err = handler.handleMessage(userID, role, ws, msg)

	// Assert - handleMessage returns nil for unknown events (doesn't break connection)
	assert.NoError(t, err)
}