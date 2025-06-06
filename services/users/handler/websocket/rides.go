package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// handleRideStartTrip processes ride start trip events from WebSocket clients
func (m *WebSocketManager) handleRideStart(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideStartRequest
	if err := json.Unmarshal(data, &event); err != nil {
		return m.SendCategorizedError(client, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
	}
	resp, err := m.userUC.RideStart(context.Background(), &event)
	if err != nil {
		return m.SendCategorizedError(client, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
	}
	// Directly notify both driver and passenger about the confirmation
	m.manager.NotifyClient(resp.DriverID.String(), constants.EventRideStarted, resp)
	m.manager.NotifyClient(resp.PassengerID.String(), constants.EventRideStarted, resp)

	return nil
}

// handleRideArrived processes ride arrival events from WebSocket clients
func (m *WebSocketManager) handleRideArrived(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideArrivalReq
	if err := json.Unmarshal(data, &event); err != nil {
		return m.SendCategorizedError(client, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
	}

	// Call the use case to process the ride arrival via HTTP
	paymentReq, err := m.userUC.RideArrived(context.Background(), &event)
	if err != nil {
		return m.SendCategorizedError(client, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
	}
	m.manager.NotifyClient(paymentReq.PassengerID, constants.EventPaymentRequest, paymentReq)

	return nil
}

// handleProcessPayment processes payment requests from WebSocket clients
func (m *WebSocketManager) handleProcessPayment(client *models.WebSocketClient, data json.RawMessage) error {
	var paymentReq models.PaymentProccessRequest
	if err := json.Unmarshal(data, &paymentReq); err != nil {
		return m.SendCategorizedError(client, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
	}
	if paymentReq.Status != models.PaymentStatusAccepted && paymentReq.Status != models.PaymentStatusRejected {
		validationErr := fmt.Errorf("invalid payment status: %s", paymentReq.Status)
		return m.SendCategorizedError(client, validationErr, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
	}

	// Call the use case to process the payment
	payment, err := m.userUC.ProcessPayment(context.Background(), &paymentReq)
	if err != nil {
		return m.SendCategorizedError(client, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
	}

	// Notify the client about the processed payment
	if err := m.manager.SendMessage(client.Conn, constants.EventPaymentProcessed, payment); err != nil {
		return err
	}

	return nil
}
