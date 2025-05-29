package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// ConfirmMatch sends a match confirmation and returns the confirmed match proposal
func (uc *UserUC) ConfirmMatch(ctx context.Context, mp *models.MatchConfirmRequest) (*models.MatchProposal, error) {
	// Validate match status
	if mp.Status != string(models.MatchStatusAccepted) &&
		mp.Status != string(models.MatchStatusRejected) {
		return nil, fmt.Errorf("invalid match status: %s", mp.Status)
	}

	// Validate that the user exists and has the correct role
	user, err := uc.userRepo.GetUserByID(ctx, mp.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	// Assign the user's role to the match proposal
	mp.Role = user.Role
	// Call the gateway with the user's role information
	return uc.UserGW.MatchConfirm(mp)
}
