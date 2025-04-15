package models

import "encoding/json"

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
