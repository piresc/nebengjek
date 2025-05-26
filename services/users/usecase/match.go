package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// ConfirmMatch sends a match confirmation and returns the confirmed match proposal
func (uc *UserUC) ConfirmMatch(ctx context.Context, mp *models.MatchProposal, userID string) (*models.MatchProposal, error) {
	if mp.MatchStatus != models.MatchStatusAccepted {
		return nil, fmt.Errorf("invalid match status: %s", mp.MatchStatus)
	}
	driver, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if driver.Role != "driver" {
		return nil, fmt.Errorf("user %s is not a driver", mp.DriverID)
	}

	return uc.UserGW.MatchAccept(mp)
}
