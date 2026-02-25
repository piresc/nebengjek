package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models/websocket"
)

// WSSessionRepository manages WebSocket sessions in Redis
type WSSessionRepository struct {
	redisClient *database.RedisClient
	serverID    string
	sessionTTL  time.Duration
}

// NewWSSessionRepository creates a new WebSocket session repository
func NewWSSessionRepository(redisClient *database.RedisClient, serverID string) *WSSessionRepository {
	return &WSSessionRepository{
		redisClient: redisClient,
		serverID:    serverID,
		sessionTTL:  5 * time.Minute, // Session expires if not updated
	}
}

// RegisterConnection registers a new WebSocket connection in Redis
func (r *WSSessionRepository) RegisterConnection(ctx context.Context, userID string, session *websocket.WSSessionData) error {
	session.ServerID = r.serverID
	
	pipe := r.redisClient.Client.Pipeline()
	
	// Store session data as hash
	sessionKey := fmt.Sprintf(database.KeyWSSession, userID)
	sessionFields := map[string]interface{}{
		database.FieldUserID:       session.UserID,
		database.FieldRole:         session.Role,
		database.FieldServerID:     session.ServerID,
		database.FieldConnectedAt:  session.ConnectedAt.Unix(),
		database.FieldLastActivity: session.LastActivity.Unix(),
		database.FieldLastPing:     session.LastPing.Unix(),
		database.FieldLastPong:     session.LastPong.Unix(),
		database.FieldMissedPings:  session.MissedPings,
		database.FieldIsHealthy:    session.IsHealthy,
	}
	
	pipe.HMSet(ctx, sessionKey, sessionFields)
	pipe.Expire(ctx, sessionKey, r.sessionTTL)
	
	// Add to global connections set
	pipe.SAdd(ctx, database.KeyWSConnections, userID)
	
	// Add to server-specific connections set
	serverConnectionsKey := fmt.Sprintf(database.KeyWSServerConnections, r.serverID)
	pipe.SAdd(ctx, serverConnectionsKey, userID)
	
	if _, err := pipe.Exec(ctx); err != nil {
		logger.ErrorCtx(ctx, "Failed to register WebSocket connection",
			logger.String("user_id", userID),
			logger.String("server_id", r.serverID),
			logger.Err(err))
		return fmt.Errorf("failed to register connection: %w", err)
	}
	
	logger.InfoCtx(ctx, "WebSocket connection registered successfully",
		logger.String("user_id", userID),
		logger.String("server_id", r.serverID))
	
	return nil
}

// UnregisterConnection removes a WebSocket connection from Redis
func (r *WSSessionRepository) UnregisterConnection(ctx context.Context, userID string) error {
	pipe := r.redisClient.Client.Pipeline()
	
	// Remove session data
	sessionKey := fmt.Sprintf(database.KeyWSSession, userID)
	pipe.Del(ctx, sessionKey)
	
	// Remove from global connections set
	pipe.SRem(ctx, database.KeyWSConnections, userID)
	
	// Remove from server-specific connections set
	serverConnectionsKey := fmt.Sprintf(database.KeyWSServerConnections, r.serverID)
	pipe.SRem(ctx, serverConnectionsKey, userID)
	
	if _, err := pipe.Exec(ctx); err != nil {
		logger.ErrorCtx(ctx, "Failed to unregister WebSocket connection",
			logger.String("user_id", userID),
			logger.String("server_id", r.serverID),
			logger.Err(err))
		return fmt.Errorf("failed to unregister connection: %w", err)
	}
	
	logger.InfoCtx(ctx, "WebSocket connection unregistered successfully",
		logger.String("user_id", userID),
		logger.String("server_id", r.serverID))
	
	return nil
}

// GetConnection retrieves WebSocket session data from Redis
func (r *WSSessionRepository) GetConnection(ctx context.Context, userID string) (*websocket.WSSessionData, error) {
	sessionKey := fmt.Sprintf(database.KeyWSSession, userID)
	
	result := r.redisClient.Client.HGetAll(ctx, sessionKey)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return nil, nil // Connection not found
		}
		return nil, fmt.Errorf("failed to get connection data: %w", err)
	}
	
	fields := result.Val()
	if len(fields) == 0 {
		return nil, nil // Connection not found
	}
	
	session := &websocket.WSSessionData{}
	
	if userID, exists := fields[database.FieldUserID]; exists {
		session.UserID = userID
	}
	if role, exists := fields[database.FieldRole]; exists {
		session.Role = role
	}
	if serverID, exists := fields[database.FieldServerID]; exists {
		session.ServerID = serverID
	}
	if connectedAt, exists := fields[database.FieldConnectedAt]; exists {
		if timestamp, err := strconv.ParseInt(connectedAt, 10, 64); err == nil {
			session.ConnectedAt = time.Unix(timestamp, 0)
		}
	}
	if lastActivity, exists := fields[database.FieldLastActivity]; exists {
		if timestamp, err := strconv.ParseInt(lastActivity, 10, 64); err == nil {
			session.LastActivity = time.Unix(timestamp, 0)
		}
	}
	if lastPing, exists := fields[database.FieldLastPing]; exists {
		if timestamp, err := strconv.ParseInt(lastPing, 10, 64); err == nil {
			session.LastPing = time.Unix(timestamp, 0)
		}
	}
	if lastPong, exists := fields[database.FieldLastPong]; exists {
		if timestamp, err := strconv.ParseInt(lastPong, 10, 64); err == nil {
			session.LastPong = time.Unix(timestamp, 0)
		}
	}
	if missedPings, exists := fields[database.FieldMissedPings]; exists {
		if count, err := strconv.Atoi(missedPings); err == nil {
			session.MissedPings = count
		}
	}
	if isHealthy, exists := fields[database.FieldIsHealthy]; exists {
		session.IsHealthy = isHealthy == "true" || isHealthy == "1"
	}
	
	return session, nil
}

// UpdateActivity updates the last activity timestamp for a connection
func (r *WSSessionRepository) UpdateActivity(ctx context.Context, userID string, activity time.Time) error {
	sessionKey := fmt.Sprintf(database.KeyWSSession, userID)
	
	pipe := r.redisClient.Client.Pipeline()
	pipe.HSet(ctx, sessionKey, database.FieldLastActivity, activity.Unix())
	pipe.Expire(ctx, sessionKey, r.sessionTTL) // Extend session TTL
	
	if _, err := pipe.Exec(ctx); err != nil {
		logger.ErrorCtx(ctx, "Failed to update activity",
			logger.String("user_id", userID),
			logger.Err(err))
		return fmt.Errorf("failed to update activity: %w", err)
	}
	
	return nil
}

// UpdateHeartbeat updates heartbeat information for a connection
func (r *WSSessionRepository) UpdateHeartbeat(ctx context.Context, userID string, ping, pong time.Time, missed int) error {
	sessionKey := fmt.Sprintf(database.KeyWSSession, userID)
	
	updates := map[string]interface{}{
		database.FieldMissedPings: missed,
		database.FieldIsHealthy:   missed < 3, // Health threshold
	}
	
	if !ping.IsZero() {
		updates[database.FieldLastPing] = ping.Unix()
	}
	if !pong.IsZero() {
		updates[database.FieldLastPong] = pong.Unix()
	}
	
	pipe := r.redisClient.Client.Pipeline()
	pipe.HMSet(ctx, sessionKey, updates)
	pipe.Expire(ctx, sessionKey, r.sessionTTL)
	
	if _, err := pipe.Exec(ctx); err != nil {
		logger.ErrorCtx(ctx, "Failed to update heartbeat",
			logger.String("user_id", userID),
			logger.Err(err))
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}
	
	return nil
}

// GetConnectedUsers returns all connected user IDs
func (r *WSSessionRepository) GetConnectedUsers(ctx context.Context) ([]string, error) {
	result := r.redisClient.Client.SMembers(ctx, database.KeyWSConnections)
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to get connected users: %w", err)
	}
	
	return result.Val(), nil
}

// GetServerConnections returns user IDs connected to a specific server
func (r *WSSessionRepository) GetServerConnections(ctx context.Context, serverID string) ([]string, error) {
	serverConnectionsKey := fmt.Sprintf(database.KeyWSServerConnections, serverID)
	
	result := r.redisClient.Client.SMembers(ctx, serverConnectionsKey)
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to get server connections: %w", err)
	}
	
	return result.Val(), nil
}

// IsUserConnected checks if a user is currently connected
func (r *WSSessionRepository) IsUserConnected(ctx context.Context, userID string) (bool, error) {
	result := r.redisClient.Client.SIsMember(ctx, database.KeyWSConnections, userID)
	if err := result.Err(); err != nil {
		return false, fmt.Errorf("failed to check connection status: %w", err)
	}
	
	return result.Val(), nil
}

// GetConnectionCount returns the total number of connected users
func (r *WSSessionRepository) GetConnectionCount(ctx context.Context) (int, error) {
	result := r.redisClient.Client.SCard(ctx, database.KeyWSConnections)
	if err := result.Err(); err != nil {
		return 0, fmt.Errorf("failed to get connection count: %w", err)
	}
	
	return int(result.Val()), nil
}