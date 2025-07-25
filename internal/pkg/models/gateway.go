package models

import "net/http"

// ProxyResponse represents a response from a microservice proxy call
type ProxyResponse struct {
	StatusCode int         `json:"status_code"`
	Body       []byte      `json:"body"`
	Headers    http.Header `json:"headers"`
}

// Notification represents a notification to be sent via WebSocket
type Notification struct {
	UserID  string      `json:"user_id"`
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
