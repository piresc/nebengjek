package websocket

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
)

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	UserID string
	Role   string
	Conn   *websocket.Conn
}

// CustomClaims represents custom JWT claims used in WebSocket authentication
type CustomClaims struct {
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
