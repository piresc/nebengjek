package models

import (
	"time"
	"encoding/json"
)

// WSBroadcastMessage represents a multi-gateway WebSocket broadcast
type WSBroadcastMessage struct {
	MessageID   string          `json:"message_id"`
	Event       string          `json:"event"`
	Data        json.RawMessage `json:"data"`
	Timestamp   time.Time       `json:"timestamp"`
	Source      string          `json:"source"`
	Priority    int             `json:"priority"`
}

// WSUserBroadcast represents a message targeted to specific users
type WSUserBroadcast struct {
	WSBroadcastMessage
	TargetUsers  []string `json:"target_users"`
	ExcludeUsers []string `json:"exclude_users,omitempty"`
}

// WSRoleBroadcast represents a message targeted to user roles
type WSRoleBroadcast struct {
	WSBroadcastMessage
	TargetRoles  []string `json:"target_roles"` // driver, passenger
	ExcludeUsers []string `json:"exclude_users,omitempty"`
}

// WSRouteMessage represents cross-server user routing
type WSRouteMessage struct {
	MessageID     string          `json:"message_id"`
	TargetUserID  string          `json:"target_user_id"`
	Event         string          `json:"event"`
	Data          json.RawMessage `json:"data"`
	Source        string          `json:"source"`
	Timestamp     time.Time       `json:"timestamp"`
	RetryCount    int             `json:"retry_count,omitempty"`
}

// WSServerInfo represents gateway server information for multi-gateway coordination
type WSServerInfo struct {
	ServerID      string    `json:"server_id"`
	Hostname      string    `json:"hostname"`
	Port          int       `json:"port"`
	Status        string    `json:"status"` // active, draining, stopped
	Connections   int       `json:"connections"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	StartTime     time.Time `json:"start_time"`
}