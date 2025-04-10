package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/services/billing"
)

// BillingHandler handles HTTP requests for billing operations
type BillingHandler struct {
	billingUC billing.BillingUseCase
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingUC billing.BillingUseCase) *BillingHandler {
	return &BillingHandler{
		billingUC: billingUC,
	}
}

// RegisterRoutes registers the billing routes
func (h *BillingHandler) RegisterRoutes(e *echo.Echo) {
	g := e.Group("/api/v1/billing")

	// Payment routes
	g.POST("/payments", h.ProcessPayment)
}
