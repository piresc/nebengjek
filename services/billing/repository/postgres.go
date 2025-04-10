package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/billing"
)

// PostgresBillingRepo implements the BillingRepo interface
type PostgresBillingRepo struct {
	db *sqlx.DB
}

// NewBillingRepository creates a new billing repository
func NewBillingRepository(db *sqlx.DB) billing.BillingRepo {
	return &PostgresBillingRepo{
		db: db,
	}
}

// CreatePayment creates a new payment record
func (r *PostgresBillingRepo) CreatePayment(ctx context.Context, payment *models.Payment) error {
	// Create a map for payment data
	paymentData := map[string]interface{}{
		"id":             payment.ID,
		"user_id":        payment.UserID,
		"amount":         payment.Amount,
		"currency":       payment.Currency,
		"status":         payment.Status,
		"start_location": payment.StartLocation,
		"end_location":   payment.EndLocation,
		"created_at":     payment.CreatedAt,
		"updated_at":     payment.UpdatedAt,
	}

	// Add completed_at if it exists
	if payment.CompletedAt != nil {
		paymentData["completed_at"] = *payment.CompletedAt
	}

	// Insert new payment record
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO payments (
			id, user_id, amount, currency, status, 
			start_location, end_location, created_at, updated_at, completed_at
		) VALUES (
			:id, :user_id, :amount, :currency, :status, 
			:start_location, :end_location, :created_at, :updated_at, :completed_at
		)
	`, paymentData)

	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// UpdatePaymentStatus updates the status of a payment
func (r *PostgresBillingRepo) UpdatePaymentStatus(ctx context.Context, paymentID string, status string) error {
	// Update payment status
	result, err := r.db.ExecContext(ctx, `
		UPDATE payments
		SET status = $1, updated_at = NOW(), 
		    completed_at = CASE WHEN $1 = 'completed' THEN NOW() ELSE completed_at END
		WHERE id = $2
	`, status, paymentID)

	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("payment not found")
	}

	return nil
}

// CreateTransaction creates a new transaction record
func (r *PostgresBillingRepo) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	// Create a map for transaction data
	transactionData := map[string]interface{}{
		"id":         transaction.ID,
		"user_id":    transaction.UserID,
		"payment_id": transaction.PaymentID,
		"amount":     transaction.Amount,
		"currency":   transaction.Currency,
		"status":     transaction.Status,
		"type":       transaction.Type,
		"created_at": transaction.CreatedAt,
		"updated_at": transaction.UpdatedAt,
	}

	// Insert new transaction record
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO transactions (
			id, user_id, payment_id, amount, currency, 
			status, type, created_at, updated_at
		) VALUES (
			:id, :user_id, :payment_id, :amount, :currency, 
			:status, :type, :created_at, :updated_at
		)
	`, transactionData)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}
