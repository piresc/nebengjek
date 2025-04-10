package billing

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// BillingRepo defines the interface for billing repository operations
type BillingRepo interface {
	CreatePayment(ctx context.Context, payment *models.Payment) error
	UpdatePaymentStatus(ctx context.Context, paymentID string, status string) error
	CreateTransaction(ctx context.Context, transaction *models.Transaction) error
}

// LocationRepo defines the interface for location operations
type LocationRepo interface {
	CalculateDistance(ctx context.Context, startLocation, endLocation string) (float64, error)
}

// BillingUseCase defines the interface for billing use cases
type BillingUseCase interface {
	ProcessPayment(ctx context.Context, payment *models.Payment) error
}
