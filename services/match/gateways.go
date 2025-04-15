package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchGW defines the match gateaways interface
type MatchGW interface {
	PublishMatchEvent(ctx context.Context, matchProp models.MatchProposal) error
}
