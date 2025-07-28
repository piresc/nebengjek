package gateway

import (
	"github.com/piresc/nebengjek/internal/pkg/database"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
	gateway_nats "github.com/piresc/nebengjek/services/match/gateway/nats"
)

// MatchGW handles match gateway operations
type MatchGW struct {
	natsGateway  *gateway_nats.NATSGateway
	redisGateway *RedisGateway
}

// NewMatchGW creates a new gateway instance with NATS and Redis clients
func NewMatchGW(natsClient *natspkg.Client, redisClient *database.RedisClient) match.MatchGW {
	return &MatchGW{
		natsGateway:  gateway_nats.NewNATSGateway(natsClient),
		redisGateway: NewRedisGateway(redisClient),
	}
}
