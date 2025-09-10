package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
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
func (r *WSSessionRepository) RegisterConnection(ctx context.Context, userID string, session *models.WSSessionData) error {
	session.ServerID = r.serverID
	
	pipe := r.redisClient.Client.Pipeline()
	
	// Store session data as hash
	sessionKey := fmt.Sprintf(constants.KeyWSSession, userID)
	sessionFields := map[string]interface{}{
		constants.FieldUserID:       session.UserID,
		constants.FieldRole:         session.Role,
		constants.FieldServerID:     session.ServerID,
		constants.FieldConnectedAt:  session.ConnectedAt.Unix(),
		constants.FieldLastActivity: session.LastActivity.Unix(),
		constants.FieldLastPing:     session.LastPing.Unix(),
		constants.FieldLastPong:     session.LastPong.Unix(),
		constants.FieldMissedPings:  session.MissedPings,
		constants.FieldIsHealthy:    session.IsHealthy,
	}
	
	pipe.HMSet(ctx, sessionKey, sessionFields)
	pipe.Expire(ctx, sessionKey, r.sessionTTL)
	
	// Add to global connections set
	pipe.SAdd(ctx, constants.KeyWSConnections, userID)
	
	// Add to server-specific connections set
	serverConnectionsKey := fmt.Sprintf(constants.KeyWSServerConnections, r.serverID)
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
	sessionKey := fmt.Sprintf(constants.KeyWSSession, userID)
	pipe.Del(ctx, sessionKey)
	
	// Remove from global connections set
	pipe.SRem(ctx, constants.KeyWSConnections, userID)
	
	// Remove from server-specific connections set
	serverConnectionsKey := fmt.Sprintf(constants.KeyWSServerConnections, r.serverID)
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
func (r *WSSessionRepository) GetConnection(ctx context.Context, userID string) (*models.WSSessionData, error) {
	sessionKey := fmt.Sprintf(constants.KeyWSSession, userID)
	
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
	
	session := &models.WSSessionData{}
	
	if userID, exists := fields[constants.FieldUserID]; exists {
		session.UserID = userID
	}
	if role, exists := fields[constants.FieldRole]; exists {
		session.Role = role
	}
	if serverID, exists := fields[constants.FieldServerID]; exists {
		session.ServerID = serverID
	}
	if connectedAt, exists := fields[constants.FieldConnectedAt]; exists {
		if timestamp, err := strconv.ParseInt(connectedAt, 10, 64); err == nil {
			session.ConnectedAt = time.Unix(timestamp, 0)
		}
	}
	if lastActivity, exists := fields[constants.FieldLastActivity]; exists {
		if timestamp, err := strconv.ParseInt(lastActivity, 10, 64); err == nil {
			session.LastActivity = time.Unix(timestamp, 0)
		}
	}
	if lastPing, exists := fields[constants.FieldLastPing]; exists {
		if timestamp, err := strconv.ParseInt(lastPing, 10, 64); err == nil {
			session.LastPing = time.Unix(timestamp, 0)
		}
	}
	if lastPong, exists := fields[constants.FieldLastPong]; exists {
		if timestamp, err := strconv.ParseInt(lastPong, 10, 64); err == nil {
			session.LastPong = time.Unix(timestamp, 0)
		}
	}
	if missedPings, exists := fields[constants.FieldMissedPings]; exists {
		if count, err := strconv.Atoi(missedPings); err == nil {
			session.MissedPings = count
		}
	}
	if isHealthy, exists := fields[constants.FieldIsHealthy]; exists {
		session.IsHealthy = isHealthy == "true" || isHealthy == "1"
	}
	
	return session, nil
}

// UpdateActivity updates the last activity timestamp for a connection
func (r *WSSessionRepository) UpdateActivity(ctx context.Context, userID string, activity time.Time) error {
	sessionKey := fmt.Sprintf(constants.KeyWSSession, userID)
	
	pipe := r.redisClient.Client.Pipeline()
	pipe.HSet(ctx, sessionKey, constants.FieldLastActivity, activity.Unix())
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
	sessionKey := fmt.Sprintf(constants.KeyWSSession, userID)
	
	updates := map[string]interface{}{
		constants.FieldMissedPings: missed,
		constants.FieldIsHealthy:   missed < 3, // Health threshold
	}
	
	if !ping.IsZero() {
		updates[constants.FieldLastPing] = ping.Unix()
	}
	if !pong.IsZero() {
		updates[constants.FieldLastPong] = pong.Unix()
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
	result := r.redisClient.Client.SMembers(ctx, constants.KeyWSConnections)
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to get connected users: %w", err)
	}
	
	return result.Val(), nil
}

// GetServerConnections returns user IDs connected to a specific server
func (r *WSSessionRepository) GetServerConnections(ctx context.Context, serverID string) ([]string, error) {
	serverConnectionsKey := fmt.Sprintf(constants.KeyWSServerConnections, serverID)
	
	result := r.redisClient.Client.SMembers(ctx, serverConnectionsKey)
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to get server connections: %w", err)
	}
	
	return result.Val(), nil
}

// IsUserConnected checks if a user is currently connected
func (r *WSSessionRepository) IsUserConnected(ctx context.Context, userID string) (bool, error) {
	result := r.redisClient.Client.SIsMember(ctx, constants.KeyWSConnections, userID)
	if err := result.Err(); err != nil {
		return false, fmt.Errorf("failed to check connection status: %w", err)
	}
	
	return result.Val(), nil
}

// GetConnectionCount returns the total number of connected users
func (r *WSSessionRepository) GetConnectionCount(ctx context.Context) (int, error) {
	result := r.redisClient.Client.SCard(ctx, constants.KeyWSConnections)
	if err := result.Err(); err != nil {
		return 0, fmt.Errorf("failed to get connection count: %w", err)
	}
	
	return int(result.Val()), nil
}

// CleanupExpiredSessions removes expired session entries from sets
func (r *WSSessionRepository) CleanupExpiredSessions(ctx context.Context) error {
	// Get all users from connections set
	connectedUsers, err := r.GetConnectedUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connected users for cleanup: %w", err)
	}
	
	pipe := r.redisClient.Client.Pipeline()
	cleanedCount := 0
	
	for _, userID := range connectedUsers {
		sessionKey := fmt.Sprintf(constants.KeyWSSession, userID)
		exists := r.redisClient.Client.Exists(ctx, sessionKey)
		
		if exists.Err() == nil && exists.Val() == 0 {
			// Session data expired but user still in sets - clean up
			pipe.SRem(ctx, constants.KeyWSConnections, userID)
			
			serverConnectionsKey := fmt.Sprintf(constants.KeyWSServerConnections, r.serverID)
			pipe.SRem(ctx, serverConnectionsKey, userID)
			
			cleanedCount++
		}
	}
	
	if cleanedCount > 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			logger.ErrorCtx(ctx, "Failed to cleanup expired sessions",
				logger.Int("cleaned_count", cleanedCount),
				logger.Err(err))
			return fmt.Errorf("failed to cleanup expired sessions: %w", err)
		}
		
		logger.InfoCtx(ctx, "Cleaned up expired WebSocket sessions",
			logger.Int("cleaned_count", cleanedCount))
	}
	
	return nil
}