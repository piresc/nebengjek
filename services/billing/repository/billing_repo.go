package repository

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

type BillingRepo interface {
	CreatePayment(ctx context.Context, payment *models.Payment) error
	UpdatePaymentStatus(ctx context.Context, paymentID string, status string) error
	CreateTransaction(ctx context.Context, transaction *models.Transaction) error
}
