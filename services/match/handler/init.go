package handler

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/match"
)

// MatchHandler handles requests for the match service
type MatchHandler struct {
	matchUC match.MatchUC
	cfg     *models.Config
}

// NewMatchHandler creates a new match handler
func NewMatchHandler(matchUC match.MatchUC, cfg *models.Config) *MatchHandler {
	return &MatchHandler{
		matchUC: matchUC,
		cfg:     cfg,
	}
}
