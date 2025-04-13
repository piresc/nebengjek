package usecase

import (
	"github.com/piresc/nebengjek/services/match"
)

// MatchUC implements the match use case interface
type MatchUC struct {
	repo match.MatchRepo
}

// NewMatchUseCase creates a new match use case
func NewMatchUC(
	repo match.MatchRepo,
) *MatchUC {
	return &MatchUC{
		repo: repo,
	}
}
