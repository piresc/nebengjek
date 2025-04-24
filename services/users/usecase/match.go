package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateBeaconStatus updates a user's beacon status and location
func (uc *UserUC) ConfirmMatch(ctx context.Context, mp *models.MatchProposal, userID string) error {
	if mp.MatchStatus != models.MatchStatusAccepted {
		return fmt.Errorf("invalid match status: %s", mp.MatchStatus)
	}
	driver, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if driver.Role != "driver" {
		return fmt.Errorf("user %s is not a driver", mp.DriverID)
	}

	return uc.UserGW.MatchAccept(mp)
}
