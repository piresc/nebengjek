package usecase

import (
	"github.com/piresc/nebengjek/services/match"
)

// MatchUC implements the match use case interface
type MatchUC struct {
	matchRepo match.MatchRepo
	matchGW   match.MatchGW
}

// NewMatchUC creates a new match use case
func NewMatchUC(
	matchRepo match.MatchRepo,
	matchGW match.MatchGW,
) *MatchUC {
	return &MatchUC{
		matchRepo: matchRepo,
		matchGW:   matchGW,
	}
}
