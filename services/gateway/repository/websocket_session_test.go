package repository

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models/websocket"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *database.RedisClient, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := &database.RedisClient{
		Client: redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		}),
	}

	cleanup := func() {
		mr.Close()
		redisClient.Client.Close()
	}

	return mr, redisClient, cleanup
}

func TestNewWSSessionRepository(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	assert.NotNil(t, repo)
	assert.Equal(t, redisClient, repo.redisClient)
	assert.Equal(t, "server-1", repo.serverID)
	assert.Equal(t, 5*time.Minute, repo.sessionTTL)
}

func TestWSSessionRepository_RegisterConnection(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		LastPing:     time.Now(),
		LastPong:     time.Now(),
		MissedPings:  0,
		IsHealthy:    true,
	}

	err := repo.RegisterConnection(ctx, "user-123", session)
	assert.NoError(t, err)

	// Verify session data was stored
	sessionKey := "ws:session:user-123"
	result := redisClient.Client.HGetAll(ctx, sessionKey)
	require.NoError(t, result.Err())
	fields := result.Val()

	assert.Equal(t, "user-123", fields[database.FieldUserID])
	assert.Equal(t, "driver", fields[database.FieldRole])
	assert.Equal(t, "server-1", fields[database.FieldServerID])
	assert.Equal(t, "1", fields[database.FieldIsHealthy])

	// Verify user is in connections sets
	assert.True(t, redisClient.Client.SIsMember(ctx, "ws:connections", "user-123").Val())
	assert.True(t, redisClient.Client.SIsMember(ctx, "ws:server:server-1:connections", "user-123").Val())

	// Verify TTL was set
	ttl := redisClient.Client.TTL(ctx, sessionKey).Val()
	assert.True(t, ttl > 0 && ttl <= 5*time.Minute)
}

func TestWSSessionRepository_RegisterConnection_WithPipelineError(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Close Redis client to simulate error
	redisClient.Client.Close()

	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		LastPing:     time.Now(),
		LastPong:     time.Now(),
		MissedPings:  0,
		IsHealthy:    true,
	}

	err := repo.RegisterConnection(ctx, "user-123", session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register connection")
}

func TestWSSessionRepository_UnregisterConnection(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// First register a connection
	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		LastPing:     time.Now(),
		LastPong:     time.Now(),
		MissedPings:  0,
		IsHealthy:    true,
	}
	err := repo.RegisterConnection(ctx, "user-123", session)
	require.NoError(t, err)

	// Now unregister it
	err = repo.UnregisterConnection(ctx, "user-123")
	assert.NoError(t, err)

	// Verify session data was removed
	sessionKey := "ws:session:user-123"
	exists := redisClient.Client.Exists(ctx, sessionKey).Val()
	assert.Equal(t, int64(0), exists)

	// Verify user was removed from connections sets
	assert.False(t, redisClient.Client.SIsMember(ctx, "ws:connections", "user-123").Val())
	assert.False(t, redisClient.Client.SIsMember(ctx, "ws:server:server-1:connections", "user-123").Val())
}

func TestWSSessionRepository_GetConnection(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// First register a connection
	now := time.Now()
	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  now,
		LastActivity: now,
		LastPing:     now,
		LastPong:     now,
		MissedPings:  0,
		IsHealthy:    true,
	}
	err := repo.RegisterConnection(ctx, "user-123", session)
	require.NoError(t, err)

	// Now retrieve it
	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.Equal(t, "driver", retrieved.Role)
	assert.Equal(t, "server-1", retrieved.ServerID)
	assert.True(t, retrieved.IsHealthy)
	assert.Equal(t, 0, retrieved.MissedPings)

	// Test with non-existent user
	retrieved, err = repo.GetConnection(ctx, "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestWSSessionRepository_GetConnection_WithPartialData(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Manually store partial session data
	sessionKey := "ws:session:user-123"
	redisClient.Client.HSet(ctx, sessionKey, map[string]interface{}{
		database.FieldUserID:     "user-123",
		database.FieldRole:       "driver",
		database.FieldIsHealthy:  "1",
	})

	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.Equal(t, "driver", retrieved.Role)
	assert.True(t, retrieved.IsHealthy)
	// Default values for missing fields
	assert.Equal(t, "", retrieved.ServerID)
	assert.Equal(t, 0, retrieved.MissedPings)
}

func TestWSSessionRepository_GetConnection_WithInvalidTimestamp(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Manually store session data with invalid timestamp
	sessionKey := "ws:session:user-123"
	redisClient.Client.HSet(ctx, sessionKey, map[string]interface{}{
		database.FieldUserID:      "user-123",
		database.FieldConnectedAt: "invalid-timestamp",
	})

	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "user-123", retrieved.UserID)
	// Invalid timestamp should result in zero time
	assert.True(t, retrieved.ConnectedAt.IsZero())
}

func TestWSSessionRepository_UpdateActivity(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// First register a connection
	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		LastPing:     time.Now(),
		LastPong:     time.Now(),
		MissedPings:  0,
		IsHealthy:    true,
	}
	err := repo.RegisterConnection(ctx, "user-123", session)
	require.NoError(t, err)

	// Update activity
	newActivity := time.Now().Add(time.Minute)
	err = repo.UpdateActivity(ctx, "user-123", newActivity)
	assert.NoError(t, err)

	// Verify activity was updated
	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.WithinDuration(t, newActivity, retrieved.LastActivity, time.Second)
}

func TestWSSessionRepository_UpdateHeartbeat(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// First register a connection
	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		LastPing:     time.Now(),
		LastPong:     time.Now(),
		MissedPings:  0,
		IsHealthy:    true,
	}
	err := repo.RegisterConnection(ctx, "user-123", session)
	require.NoError(t, err)

	// Update heartbeat
	pingTime := time.Now().Add(time.Minute)
	pongTime := time.Now().Add(time.Minute + 30*time.Second)
	err = repo.UpdateHeartbeat(ctx, "user-123", pingTime, pongTime, 2)
	assert.NoError(t, err)

	// Verify heartbeat was updated
	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.WithinDuration(t, pingTime, retrieved.LastPing, time.Second)
	assert.WithinDuration(t, pongTime, retrieved.LastPong, time.Second)
	assert.Equal(t, 2, retrieved.MissedPings)
	assert.True(t, retrieved.IsHealthy) // 2 < 3, so should be healthy

	// Test with unhealthy connection (missed pings >= 3)
	err = repo.UpdateHeartbeat(ctx, "user-123", pingTime, pongTime, 3)
	assert.NoError(t, err)

	retrieved, err = repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.Equal(t, 3, retrieved.MissedPings)
	assert.False(t, retrieved.IsHealthy) // 3 >= 3, so should be unhealthy
}

func TestWSSessionRepository_UpdateHeartbeat_ZeroTimes(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// First register a connection
	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		LastPing:     time.Now(),
		LastPong:     time.Now(),
		MissedPings:  0,
		IsHealthy:    true,
	}
	err := repo.RegisterConnection(ctx, "user-123", session)
	require.NoError(t, err)

	// Update heartbeat with zero times (should not update ping/pong fields)
	err = repo.UpdateHeartbeat(ctx, "user-123", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)

	// Verify ping/pong times were not updated
	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.Equal(t, 1, retrieved.MissedPings)
	// Original ping/pong times should be preserved
	assert.False(t, retrieved.LastPing.IsZero())
	assert.False(t, retrieved.LastPong.IsZero())
}

func TestWSSessionRepository_GetConnectedUsers(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Register multiple connections
	users := []string{"user-1", "user-2", "user-3"}
	for _, userID := range users {
		session := &websocket.WSSessionData{
			UserID:       userID,
			Role:         "driver",
			ConnectedAt:  time.Now(),
			LastActivity: time.Now(),
			LastPing:     time.Now(),
			LastPong:     time.Now(),
			MissedPings:  0,
			IsHealthy:    true,
		}
		err := repo.RegisterConnection(ctx, userID, session)
		require.NoError(t, err)
	}

	// Get connected users
	connectedUsers, err := repo.GetConnectedUsers(ctx)
	assert.NoError(t, err)
	assert.Len(t, connectedUsers, 3)
	assert.Contains(t, connectedUsers, "user-1")
	assert.Contains(t, connectedUsers, "user-2")
	assert.Contains(t, connectedUsers, "user-3")
}

func TestWSSessionRepository_GetServerConnections(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Register connections
	users := []string{"user-1", "user-2"}
	for _, userID := range users {
		session := &websocket.WSSessionData{
			UserID:       userID,
			Role:         "driver",
			ConnectedAt:  time.Now(),
			LastActivity: time.Now(),
			LastPing:     time.Now(),
			LastPong:     time.Now(),
			MissedPings:  0,
			IsHealthy:    true,
		}
		err := repo.RegisterConnection(ctx, userID, session)
		require.NoError(t, err)
	}

	// Get server connections
	serverConnections, err := repo.GetServerConnections(ctx, "server-1")
	assert.NoError(t, err)
	assert.Len(t, serverConnections, 2)
	assert.Contains(t, serverConnections, "user-1")
	assert.Contains(t, serverConnections, "user-2")

	// Test non-existent server
	emptyConnections, err := repo.GetServerConnections(ctx, "non-existent-server")
	assert.NoError(t, err)
	assert.Empty(t, emptyConnections)
}

func TestWSSessionRepository_IsUserConnected(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Initially user should not be connected
	connected, err := repo.IsUserConnected(ctx, "user-123")
	assert.NoError(t, err)
	assert.False(t, connected)

	// Register connection
	session := &websocket.WSSessionData{
		UserID:       "user-123",
		Role:         "driver",
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		LastPing:     time.Now(),
		LastPong:     time.Now(),
		MissedPings:  0,
		IsHealthy:    true,
	}
	err = repo.RegisterConnection(ctx, "user-123", session)
	require.NoError(t, err)

	// Now user should be connected
	connected, err = repo.IsUserConnected(ctx, "user-123")
	assert.NoError(t, err)
	assert.True(t, connected)
}

func TestWSSessionRepository_GetConnectionCount(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Initially no connections
	count, err := repo.GetConnectionCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// Register connections
	users := []string{"user-1", "user-2", "user-3"}
	for _, userID := range users {
		session := &websocket.WSSessionData{
			UserID:       userID,
			Role:         "driver",
			ConnectedAt:  time.Now(),
			LastActivity: time.Now(),
			LastPing:     time.Now(),
			LastPong:     time.Now(),
			MissedPings:  0,
			IsHealthy:    true,
		}
		err := repo.RegisterConnection(ctx, userID, session)
		require.NoError(t, err)
	}

	// Check count
	count, err = repo.GetConnectionCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestWSSessionRepository_InvalidTimestampParsing(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Manually store session data with invalid numeric timestamp
	sessionKey := "ws:session:user-123"
	redisClient.Client.HSet(ctx, sessionKey, map[string]interface{}{
		database.FieldUserID:      "user-123",
		database.FieldConnectedAt: "not-a-number",
		database.FieldMissedPings: "not-a-number",
	})

	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.True(t, retrieved.ConnectedAt.IsZero()) // Invalid timestamp should be zero
	assert.Equal(t, 0, retrieved.MissedPings)     // Invalid number should be 0
}

func TestWSSessionRepository_InvalidBooleanParsing(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Manually store session data with invalid boolean
	sessionKey := "ws:session:user-123"
	redisClient.Client.HSet(ctx, sessionKey, map[string]interface{}{
		database.FieldUserID:     "user-123",
		database.FieldIsHealthy: "not-a-boolean",
	})

	retrieved, err := repo.GetConnection(ctx, "user-123")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.False(t, retrieved.IsHealthy) // Invalid boolean should be false
}

func TestWSSessionRepository_ConcurrentAccess(t *testing.T) {
	_, redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := NewWSSessionRepository(redisClient, "server-1")
	ctx := context.Background()

	// Test concurrent access to the same user
	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			userID := "user-" + strconv.Itoa(id)
			session := &websocket.WSSessionData{
				UserID:       userID,
				Role:         "driver",
				ConnectedAt:  time.Now(),
				LastActivity: time.Now(),
				LastPing:     time.Now(),
				LastPong:     time.Now(),
				MissedPings:  0,
				IsHealthy:    true,
			}
			_ = repo.RegisterConnection(ctx, userID, session)
		}(i)
	}

	wg.Wait()

	// Verify all users were registered
	connectedUsers, err := repo.GetConnectedUsers(ctx)
	assert.NoError(t, err)
	assert.Len(t, connectedUsers, numGoroutines)
}