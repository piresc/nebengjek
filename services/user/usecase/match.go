package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateBeaconStatus updates a user's beacon status and location
func (uc *UserUC) ConfirmMatch(ctx context.Context, mp *models.MatchProposal) error {
	if mp.MatchStatus != models.MatchStatusAccepted {
		return fmt.Errorf("invalid match status: %s", mp.MatchStatus)
	}
	return uc.UserGW.MatchAccept(mp)
}
