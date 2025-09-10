package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// WebSocketClaims represents custom JWT claims used in WebSocket authentication
type WebSocketClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	MSISDN string `json:"msisdn"`
}

// WSMessage represents a WebSocket message structure
type WSMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// WSErrorMessage represents an error message sent over WebSocket
type WSErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WSHeartbeatMessage represents a heartbeat ping message (coder/websocket handles pong automatically)
type WSHeartbeatMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Sequence  int64     `json:"sequence,omitempty"`
}

// WSConnectionHealth represents connection health status using coder/websocket capabilities
type WSConnectionHealth struct {
	UserID         string    `json:"user_id"`
	LastActivity   time.Time `json:"last_activity"`
	LastPing       time.Time `json:"last_ping"`
	MissedPings    int       `json:"missed_pings"`
	IsHealthy      bool      `json:"is_healthy"`
	ConnectedSince time.Time `json:"connected_since"`
}

// WSSessionData represents WebSocket session data stored in Redis
type WSSessionData struct {
	UserID       string    `json:"user_id" redis:"user_id"`
	Role         string    `json:"role" redis:"role"`
	ServerID     string    `json:"server_id" redis:"server_id"`
	ConnectedAt  time.Time `json:"connected_at" redis:"connected_at"`
	LastActivity time.Time `json:"last_activity" redis:"last_activity"`
	LastPing     time.Time `json:"last_ping" redis:"last_ping"`
	LastPong     time.Time `json:"last_pong" redis:"last_pong"`
	MissedPings  int       `json:"missed_pings" redis:"missed_pings"`
	IsHealthy    bool      `json:"is_healthy" redis:"is_healthy"`
}

// WSConnectionRegistry manages distributed connection registry
type WSConnectionRegistry interface {
	RegisterConnection(ctx context.Context, userID string, session *WSSessionData) error
	UnregisterConnection(ctx context.Context, userID string) error
	GetConnection(ctx context.Context, userID string) (*WSSessionData, error)
	UpdateActivity(ctx context.Context, userID string, activity time.Time) error
	UpdateHeartbeat(ctx context.Context, userID string, ping, pong time.Time, missed int) error
	GetConnectedUsers(ctx context.Context) ([]string, error)
	GetServerConnections(ctx context.Context, serverID string) ([]string, error)
	IsUserConnected(ctx context.Context, userID string) (bool, error)
	GetConnectionCount(ctx context.Context) (int, error)
}
