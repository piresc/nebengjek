package usecase

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/match"
)

// MatchUC implements the match use case interface
type MatchUC struct {
	matchRepo match.MatchRepo
	matchGW   match.MatchGW
	cfg       *models.Config
}

// NewMatchUC creates a new match use case
func NewMatchUC(
	cfg *models.Config,
	matchRepo match.MatchRepo,
	matchGW match.MatchGW,
) *MatchUC {
	return &MatchUC{
		cfg:       cfg,
		matchRepo: matchRepo,
		matchGW:   matchGW,
	}
}
