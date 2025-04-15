package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchGW defines the match gateaways interface
type MatchGW interface {
	PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error
	PublishMatchAccept(ctx context.Context, matchProp models.MatchProposal) error
	PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error
}
