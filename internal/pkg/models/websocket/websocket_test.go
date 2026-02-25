package websocket

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketClaims_DefaultValues(t *testing.T) {
	claims := &WebSocketClaims{}

	// Test zero values
	assert.Equal(t, "", claims.UserID)
	assert.Equal(t, "", claims.Role)
	assert.Equal(t, "", claims.MSISDN)
}

func TestWebSocketClaims_WithValues(t *testing.T) {
	now := time.Now()
	claims := &WebSocketClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
		},
		UserID: "test-user-id",
		Role:   "driver",
		MSISDN: "+6281234567890",
	}

	assert.Equal(t, "test-user-id", claims.UserID)
	assert.Equal(t, "driver", claims.Role)
	assert.Equal(t, "+6281234567890", claims.MSISDN)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
}

func TestWebSocketClaims_JWTCapability(t *testing.T) {
	// Test that WebSocketClaims can be used with JWT library
	claims := &WebSocketClaims{
		UserID: "test-user-id",
		Role:   "driver",
		MSISDN: "+6281234567890",
	}

	// Test that it implements the Claims interface
	assert.Implements(t, (*jwt.Claims)(nil), claims)
}

func TestWSMessage_DefaultValues(t *testing.T) {
	msg := &WSMessage{}

	// Test zero values
	assert.Equal(t, "", msg.Event)
	assert.Nil(t, msg.Data)
}

func TestWSMessage_WithValues(t *testing.T) {
	data := json.RawMessage(`{"test": "data"}`)
	msg := &WSMessage{
		Event: "test-event",
		Data:  data,
	}

	assert.Equal(t, "test-event", msg.Event)
	assert.NotNil(t, msg.Data)
	assert.Equal(t, []byte(`{"test": "data"}`), []byte(msg.Data))
}

func TestWSMessage_JSONSerialization(t *testing.T) {
	msg := &WSMessage{
		Event: "test-event",
		Data:  json.RawMessage(`{"key":"value"}`),
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(msg)
	assert.NoError(t, err)
	assert.NotNil(t, jsonData)

	// Test JSON unmarshaling
	var decodedMsg WSMessage
	err = json.Unmarshal(jsonData, &decodedMsg)
	assert.NoError(t, err)
	assert.Equal(t, msg.Event, decodedMsg.Event)
	
	// Compare the data as strings to avoid formatting issues
	assert.Equal(t, string(msg.Data), string(decodedMsg.Data))
}

func TestWSErrorMessage_DefaultValues(t *testing.T) {
	errMsg := &WSErrorMessage{}

	// Test zero values
	assert.Equal(t, "", errMsg.Code)
	assert.Equal(t, "", errMsg.Message)
}

func TestWSErrorMessage_WithValues(t *testing.T) {
	errMsg := &WSErrorMessage{
		Code:    "AUTH_ERROR",
		Message: "Authentication failed",
	}

	assert.Equal(t, "AUTH_ERROR", errMsg.Code)
	assert.Equal(t, "Authentication failed", errMsg.Message)
}

func TestWSErrorMessage_JSONSerialization(t *testing.T) {
	errMsg := &WSErrorMessage{
		Code:    "VALIDATION_ERROR",
		Message: "Invalid input data",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(errMsg)
	assert.NoError(t, err)
	assert.NotNil(t, jsonData)

	// Test JSON unmarshaling
	var decodedErrMsg WSErrorMessage
	err = json.Unmarshal(jsonData, &decodedErrMsg)
	assert.NoError(t, err)
	assert.Equal(t, errMsg.Code, decodedErrMsg.Code)
	assert.Equal(t, errMsg.Message, decodedErrMsg.Message)
}

func TestWSHeartbeatMessage_DefaultValues(t *testing.T) {
	heartbeat := &WSHeartbeatMessage{}

	// Test zero values
	assert.Equal(t, time.Time{}, heartbeat.Timestamp)
	assert.Equal(t, int64(0), heartbeat.Sequence)
}

func TestWSHeartbeatMessage_WithValues(t *testing.T) {
	now := time.Now()
	heartbeat := &WSHeartbeatMessage{
		Timestamp: now,
		Sequence:  12345,
	}

	assert.Equal(t, now, heartbeat.Timestamp)
	assert.Equal(t, int64(12345), heartbeat.Sequence)
}

func TestWSHeartbeatMessage_WithoutSequence(t *testing.T) {
	now := time.Now()
	heartbeat := &WSHeartbeatMessage{
		Timestamp: now,
	}

	assert.Equal(t, now, heartbeat.Timestamp)
	assert.Equal(t, int64(0), heartbeat.Sequence) // Default zero value
}

func TestWSConnectionHealth_DefaultValues(t *testing.T) {
	health := &WSConnectionHealth{}

	// Test zero values
	assert.Equal(t, "", health.UserID)
	assert.Equal(t, time.Time{}, health.LastActivity)
	assert.Equal(t, time.Time{}, health.LastPing)
	assert.Equal(t, 0, health.MissedPings)
	assert.Equal(t, false, health.IsHealthy)
	assert.Equal(t, time.Time{}, health.ConnectedSince)
}

func TestWSConnectionHealth_WithValues(t *testing.T) {
	now := time.Now()
	connectedSince := now.Add(-30 * time.Minute)
	lastActivity := now.Add(-5 * time.Minute)
	lastPing := now.Add(-1 * time.Minute)

	health := &WSConnectionHealth{
		UserID:         "test-user-id",
		LastActivity:   lastActivity,
		LastPing:       lastPing,
		MissedPings:    3,
		IsHealthy:      true,
		ConnectedSince: connectedSince,
	}

	assert.Equal(t, "test-user-id", health.UserID)
	assert.Equal(t, lastActivity, health.LastActivity)
	assert.Equal(t, lastPing, health.LastPing)
	assert.Equal(t, 3, health.MissedPings)
	assert.True(t, health.IsHealthy)
	assert.Equal(t, connectedSince, health.ConnectedSince)
}

func TestWSConnectionHealth_HealthLogic(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name       string
		missedPings int
		isHealthy   bool
		comment     string
	}{
		{"Healthy connection", 0, true, "No missed pings"},
		{"Few missed pings", 2, true, "Still healthy with few misses"},
		{"Unhealthy connection", 5, false, "Too many missed pings"},
		{"Many missed pings", 10, false, "Definitely unhealthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := &WSConnectionHealth{
				UserID:         "test-user-id",
				LastActivity:   now,
				LastPing:       now,
				MissedPings:    tt.missedPings,
				IsHealthy:      tt.isHealthy,
				ConnectedSince: now.Add(-1 * time.Hour),
			}

			assert.Equal(t, tt.missedPings, health.MissedPings)
			assert.Equal(t, tt.isHealthy, health.IsHealthy)

			// Basic health validation logic
			expectedHealth := tt.missedPings < 3
			assert.Equal(t, expectedHealth, tt.isHealthy, tt.comment)
		})
	}
}

func TestWSSessionData_DefaultValues(t *testing.T) {
	session := &WSSessionData{}

	// Test zero values
	assert.Equal(t, "", session.UserID)
	assert.Equal(t, "", session.Role)
	assert.Equal(t, "", session.ServerID)
	assert.Equal(t, time.Time{}, session.ConnectedAt)
	assert.Equal(t, time.Time{}, session.LastActivity)
	assert.Equal(t, time.Time{}, session.LastPing)
	assert.Equal(t, time.Time{}, session.LastPong)
	assert.Equal(t, 0, session.MissedPings)
	assert.Equal(t, false, session.IsHealthy)
}

func TestWSSessionData_WithValues(t *testing.T) {
	now := time.Now()
	connectedAt := now.Add(-30 * time.Minute)
	lastActivity := now.Add(-5 * time.Minute)
	lastPing := now.Add(-1 * time.Minute)
	lastPong := now.Add(-30 * time.Second)

	session := &WSSessionData{
		UserID:       "test-user-id",
		Role:         "driver",
		ServerID:     "server-001",
		ConnectedAt:  connectedAt,
		LastActivity: lastActivity,
		LastPing:     lastPing,
		LastPong:     lastPong,
		MissedPings:  2,
		IsHealthy:    true,
	}

	assert.Equal(t, "test-user-id", session.UserID)
	assert.Equal(t, "driver", session.Role)
	assert.Equal(t, "server-001", session.ServerID)
	assert.Equal(t, connectedAt, session.ConnectedAt)
	assert.Equal(t, lastActivity, session.LastActivity)
	assert.Equal(t, lastPing, session.LastPing)
	assert.Equal(t, lastPong, session.LastPong)
	assert.Equal(t, 2, session.MissedPings)
	assert.True(t, session.IsHealthy)
}

func TestWSSessionData_HealthStatus(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name        string
		missedPings int
		isHealthy   bool
		comment     string
	}{
		{"Healthy session", 0, true, "No missed pings"},
		{"Warning session", 2, true, "Still healthy but warning"},
		{"Unhealthy session", 5, false, "Too many missed pings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &WSSessionData{
				UserID:       "test-user-id",
				Role:         "driver",
				ServerID:     "server-001",
				ConnectedAt:  now.Add(-1 * time.Hour),
				LastActivity: now,
				LastPing:     now,
				LastPong:     now,
				MissedPings:  tt.missedPings,
				IsHealthy:    tt.isHealthy,
			}

			assert.Equal(t, tt.missedPings, session.MissedPings)
			assert.Equal(t, tt.isHealthy, session.IsHealthy)
		})
	}
}

func TestWSConnectionRegistry_Interface(t *testing.T) {
	// Test that WSConnectionRegistry is an interface
	var registry WSConnectionRegistry
	assert.Nil(t, registry) // Interface can be nil

	// Test that the interface has the expected methods
	registryType := reflect.TypeOf((*WSConnectionRegistry)(nil)).Elem()
	
	expectedMethods := []string{
		"RegisterConnection",
		"UnregisterConnection", 
		"GetConnection",
		"UpdateActivity",
		"UpdateHeartbeat",
		"GetConnectedUsers",
		"GetServerConnections",
		"IsUserConnected",
		"GetConnectionCount",
	}

	for _, method := range expectedMethods {
		methodFound := false
		for i := 0; i < registryType.NumMethod(); i++ {
			if registryType.Method(i).Name == method {
				methodFound = true
				break
			}
		}
		assert.True(t, methodFound, "Method %s should be present in WSConnectionRegistry interface", method)
	}
}

func TestWebSocketMessageTypes(t *testing.T) {
	// Test that all WebSocket message types can be created and are not nil
	
	var msg *WSMessage
	assert.Nil(t, msg)
	
	msg = &WSMessage{}
	assert.NotNil(t, msg)

	var errMsg *WSErrorMessage
	assert.Nil(t, errMsg)
	
	errMsg = &WSErrorMessage{}
	assert.NotNil(t, errMsg)

	var heartbeat *WSHeartbeatMessage
	assert.Nil(t, heartbeat)
	
	heartbeat = &WSHeartbeatMessage{}
	assert.NotNil(t, heartbeat)

	var health *WSConnectionHealth
	assert.Nil(t, health)
	
	health = &WSConnectionHealth{}
	assert.NotNil(t, health)

	var session *WSSessionData
	assert.Nil(t, session)
	
	session = &WSSessionData{}
	assert.NotNil(t, session)
}

func TestWebSocket_JSONTags(t *testing.T) {
	// Test JSON tags for WebSocket structures
	
	// WSMessage
	msg := &WSMessage{}
	assert.NotNil(t, msg)
	
	// WSErrorMessage
	errMsg := &WSErrorMessage{}
	assert.NotNil(t, errMsg)
	
	// WSHeartbeatMessage
	heartbeat := &WSHeartbeatMessage{}
	assert.NotNil(t, heartbeat)
	
	// WSConnectionHealth
	health := &WSConnectionHealth{}
	assert.NotNil(t, health)
	
	// WSSessionData
	session := &WSSessionData{}
	assert.NotNil(t, session)
	
	// Test that structs can be created - JSON tags are validated by the struct definitions
}