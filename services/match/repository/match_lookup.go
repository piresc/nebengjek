package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// GetPendingMatchByID retrieves a pending match from Redis by match ID
func (r *MatchRepo) GetPendingMatchByID(ctx context.Context, matchID string) (*models.Match, error) {
	// First, get the match proposal directly
	key := fmt.Sprintf(constants.KeyMatchProposal, matchID)
	fmt.Printf("Looking for pending match with ID %s in Redis using key: %s\n", matchID, key)

	matchData, err := r.redisClient.Get(ctx, key)
	if err == nil {
		// Match found directly by ID
		var match models.Match
		if err := json.Unmarshal([]byte(matchData), &match); err != nil {
			fmt.Printf("Failed to unmarshal match data: %v, data: %s\n", err, matchData)
			return nil, fmt.Errorf("failed to unmarshal match data: %w", err)
		}

		// Ensure match ID is set properly
		if match.ID == uuid.Nil {
			match.ID, _ = uuid.Parse(matchID)
			fmt.Printf("Set nil match ID to the Redis key ID: %s\n", matchID)
		}

		fmt.Printf("Found pending match in Redis: %+v\n", match)
		return &match, nil
	}

	fmt.Printf("Match not found in Redis with key %s, error: %v\n", key, err)
	return nil, fmt.Errorf("pending match with ID %s not found", matchID)
}
