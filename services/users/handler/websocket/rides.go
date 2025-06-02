package websocket

import (
	"context"
	"encoding/json"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// handleRideStartTrip processes ride start trip events from WebSocket clients
func (m *WebSocketManager) handleRideStart(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideStartRequest
	if err := json.Unmarshal(data, &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride start trip format")
	}
	resp, err := m.userUC.RideStart(context.Background(), &event)
	if err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process ride start trip")
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
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride arrival format")
	}

	// Call the use case to process the ride arrival via HTTP
	paymentReq, err := m.userUC.RideArrived(context.Background(), &event)
	if err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process ride arrival")
	}
	m.manager.NotifyClient(paymentReq.PassengerID, constants.EventPaymentRequest, paymentReq)

	return nil
}

// handleProcessPayment processes payment requests from WebSocket clients
func (m *WebSocketManager) handleProcessPayment(client *models.WebSocketClient, data json.RawMessage) error {
	var paymentReq models.PaymentRequest
	if err := json.Unmarshal(data, &paymentReq); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid payment request format")
	}

	// Call the use case to process the payment
	payment, err := m.userUC.ProcessPayment(context.Background(), &paymentReq)
	if err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process payment")
	}

	// Notify the client about the processed payment
	if err := m.manager.SendMessage(client.Conn, constants.EventPaymentProcessed, payment); err != nil {
		return err
	}

	return nil
}
