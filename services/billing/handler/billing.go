package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// ProcessPayment handles payment processing requests
func (h *BillingHandler) ProcessPayment(c echo.Context) error {
	var payment models.Payment
	if err := c.Bind(&payment); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Process the payment
	if err := h.billingUC.ProcessPayment(c.Request().Context(), &payment); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, payment)
}
