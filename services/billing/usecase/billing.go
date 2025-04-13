package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/billing"
)

// BillingUC implements the billing.BillingUseCase interface
type BillingUC struct {
	cfg        *models.Config
	repo       billing.BillingRepo
	natsClient *nats.Conn
}

// NewBillingUC creates a new billing use case
func NewBillingUC(cfg *models.Config, repo billing.BillingRepo, nc *nats.Conn) billing.BillingUseCase {
	return &BillingUC{
		cfg:        cfg,
		repo:       repo,
		natsClient: nc,
	}
}

// ProcessPayment processes a payment
func (uc *BillingUC) ProcessPayment(ctx context.Context, payment *models.Payment) error {
	// Validate payment data
	if payment.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if payment.StartLocation == "" || payment.EndLocation == "" {
		return fmt.Errorf("start and end locations are required")
	}
	distance := 10.0 // Replace with actual distance calculation logic
	// Calculate fare with 5% admin fee deduction
	baseFare := distance * uc.cfg.Pricing.RatePerKm
	payment.Amount = baseFare * 0.95

	if payment.Amount <= 0 {
		return fmt.Errorf("calculated payment amount must be positive")
	}

	// Set default values
	if payment.ID == "" {
		payment.ID = uuid.New().String()
	}
	if payment.Status == "" {
		payment.Status = "pending"
	}
	if payment.Currency == "" {
		payment.Currency = uc.cfg.Pricing.Currency
	}
	now := time.Now()
	payment.CreatedAt = now
	payment.UpdatedAt = now

	// Create payment record
	if err := uc.repo.CreatePayment(ctx, payment); err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	// Process payment with payment gateway (simplified)
	// In a real implementation, this would integrate with a payment gateway
	// and handle asynchronous callbacks

	// For demo purposes, we'll just mark it as completed
	payment.Status = "completed"
	payment.CompletedAt = &now
	payment.UpdatedAt = now

	// Update payment status
	if err := uc.repo.UpdatePaymentStatus(ctx, payment.ID, payment.Status); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Create transaction record
	transaction := &models.Transaction{
		ID:        uuid.New().String(),
		UserID:    payment.UserID,
		PaymentID: payment.ID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Status:    "completed",
		Type:      "payment",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create transaction record
	if err := uc.repo.CreateTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Publish payment event
	event := models.PaymentEvent{
		ID:        payment.ID,
		UserID:    payment.UserID,
		Amount:    payment.Amount,
		Status:    payment.Status,
		Timestamp: time.Now().UTC(),
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal payment event: %w", err)
	}

	if err := uc.natsClient.Publish("payments.processed", eventData); err != nil {
		return fmt.Errorf("failed to publish payment event: %w", err)
	}

	return nil
}
