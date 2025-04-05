package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/piresc/nebengjek/match-service/domain/entity"
)

const (
	matchKeyPrefix = "match:"
	matchTTL       = 24 * time.Hour // Store match data for 24 hours
)

type matchRepository struct {
	client *redis.Client
}

func NewMatchRepository(client *redis.Client) *matchRepository {
	return &matchRepository{
		client: client,
	}
}

func (r *matchRepository) Create(ctx context.Context, match *entity.Match) error {
	data, err := json.Marshal(match)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, matchKeyPrefix+match.ID, data, matchTTL).Err()
}

func (r *matchRepository) UpdateStatus(ctx context.Context, matchID string, status string) error {
	txf := func(tx *redis.Tx) error {
		// Get the current match data
		match, err := r.getFromTx(tx, matchID)
		if err != nil {
			return err
		}

		// Update status
		match.Status = status
		data, err := json.Marshal(match)
		if err != nil {
			return err
		}

		// Update the match with the same TTL
		return tx.Set(ctx, matchKeyPrefix+matchID, data, matchTTL).Err()
	}

	key := matchKeyPrefix + matchID
	for i := 0; i < 3; i++ { // Retry up to 3 times
		err := r.client.Watch(ctx, txf, key)
		if err != redis.TxFailedErr {
			return err
		}
	}

	return redis.TxFailedErr
}

func (r *matchRepository) Get(ctx context.Context, matchID string) (*entity.Match, error) {
	data, err := r.client.Get(ctx, matchKeyPrefix+matchID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	var match entity.Match
	if err := json.Unmarshal(data, &match); err != nil {
		return nil, err
	}

	return &match, nil
}

// Helper function to get match data within a transaction
func (r *matchRepository) getFromTx(tx *redis.Tx, matchID string) (*entity.Match, error) {
	data, err := tx.Get(tx.Context(), matchKeyPrefix+matchID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	var match entity.Match
	if err := json.Unmarshal(data, &match); err != nil {
		return nil, err
	}

	return &match, nil
}
