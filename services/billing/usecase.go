package billing

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// BillingUseCase defines the interface for billing use cases
type BillingUseCase interface {
	ProcessPayment(ctx context.Context, payment *models.Payment) error
}
