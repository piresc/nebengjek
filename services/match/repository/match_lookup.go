package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// GetPendingMatchByID retrieves a pending match from Redis by match ID
func (r *MatchRepo) GetPendingMatchByID(ctx context.Context, matchID string) (*models.Match, error) {
	// First, get the match proposal directly
	key := fmt.Sprintf(constants.KeyMatchProposal, matchID)
	matchData, err := r.redisClient.Get(ctx, key)
	if err == nil {
		// Match found directly by ID
		var match models.Match
		if err := json.Unmarshal([]byte(matchData), &match); err != nil {
			return nil, fmt.Errorf("failed to unmarshal match data: %w", err)
		}
		return &match, nil
	}

	return nil, fmt.Errorf("pending match with ID %s not found", matchID)
}
