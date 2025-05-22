package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/converter"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// ClientCustomerMatchResponse defines the structure for the 'client.customer.match_response' WebSocket message.
type ClientCustomerMatchResponse struct {
	MatchID   string `json:"matchID"`
	Confirmed bool   `json:"confirmed"`
	DriverID  string `json:"driverID"` // Client needs to send this back.
	// Consider adding UserLocation and DriverLocation if available and needed for NATS message.
}

// handleBeaconUpdate processes beacon status updates from clients
func (m *WebSocketManager) handleBeaconUpdate(client *models.WebSocketClient, data json.RawMessage) error {
	var req models.BeaconRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid beacon request format")
	}

	// Update beacon status
	if err := m.userUC.UpdateBeaconStatus(context.Background(), &req); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidBeacon, err.Error())
	}

	return m.manager.SendMessage(client.Conn, constants.EventBeaconUpdate, models.BeaconResponse{
		Message: "Beacon status updated successfully",
	})
}

// handleMatchAccept processes match acceptance from drivers
func (m *WebSocketManager) handleMatchAccept(client *models.WebSocketClient, data json.RawMessage) error {
	UserID := client.UserID

	var matchProposalAccept models.MatchProposal
	if err := json.Unmarshal(data, &matchProposalAccept); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid match proposal format")
	}

	// Update match status
	err := m.userUC.ConfirmMatch(context.Background(), &matchProposalAccept, UserID)
	if err != nil {
		log.Printf("Error confirming match for driver %s: %v", client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorMatchUpdateFailed, err.Error())
	}
	return nil
}

// handleLocationUpdate processes location updates from clients
func (m *WebSocketManager) handleLocationUpdate(driverID string, data json.RawMessage) error {
	var locationUpdate models.LocationUpdate
	if err := json.Unmarshal(data, &locationUpdate); err != nil {
		log.Printf("Error parsing location update from user %s: %v", driverID, err)
		return fmt.Errorf("invalid location format")
	}

	log.Printf("Location update from user %s: lat=%f, lng=%f, tripID=%s",
		driverID, locationUpdate.Location.Latitude, locationUpdate.Location.Longitude, locationUpdate.RideID)

	// Set timestamp if not provided
	if locationUpdate.Location.Timestamp.IsZero() {
		locationUpdate.Location.Timestamp = time.Now()
	}
	locationUpdate.DriverID = driverID
	locationUpdate.CreatedAt = time.Now()

	// Forward location update to the user usecase
	if err := m.userUC.UpdateUserLocation(context.Background(), &locationUpdate); err != nil {
		client, exists := m.manager.GetClient(driverID)
		if !exists {
			log.Printf("Client with ID %s not found", driverID)
			return fmt.Errorf("client not found")
		}
		log.Printf("Error updating location for user %s: %v", driverID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidLocation, err.Error())
	}

	return nil
}

// handleCustomerMatchResponse processes the customer's response (acceptance/rejection) to a match confirmation request.
func (m *WebSocketManager) handleCustomerMatchResponse(client *models.WebSocketClient, data json.RawMessage) error {
	ctx := context.Background()
	var req ClientCustomerMatchResponse

	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("Error unmarshalling customer match response from user %s: %v", client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid customer match response format")
	}

	log.Printf("Received customer match response from %s for match %s. Confirmed: %t", client.UserID, req.MatchID, req.Confirmed)

	// Prepare the MatchProposal for NATS publication.
	// PassengerID comes from the WebSocket client's session (client.UserID).
	// DriverID is assumed to be sent back by the client in the WebSocket message (req.DriverID).
	// Locations are omitted for now as per priority. If they were part of req or fetched from a cache,
	// they would be populated here. models.MatchProposal uses string IDs.
	matchProposal := &models.MatchProposal{
		ID:          req.MatchID,
		PassengerID: client.UserID,
		DriverID:    req.DriverID,
		// UserLocation and DriverLocation will be zero values.
	}

	var natsSubject string
	if req.Confirmed {
		matchProposal.MatchStatus = models.MatchStatusAccepted // Customer accepts
		natsSubject = constants.SubjectCustomerMatchConfirmed
	} else {
		matchProposal.MatchStatus = models.MatchStatusRejected // Customer rejects
		natsSubject = constants.SubjectCustomerMatchRejected
	}

	// Delegate to the user usecase to handle the business logic and NATS publishing.
	// This is consistent with other handlers like handleMatchAccept.
	// The userUC.HandleCustomerMatchDecision method would be responsible for
	// marshalling the matchProposal and publishing it to the natsSubject.
	// The actual implementation of UserUsecase.HandleCustomerMatchDecision is
	// not part of this specific subtask.
	log.Printf("Calling UserUsecase.HandleCustomerMatchDecision for match %s, UserID %s, Subject: %s",
		req.MatchID, client.UserID, natsSubject)

	err := m.userUC.HandleCustomerMatchDecision(ctx, matchProposal, natsSubject)
	if err != nil {
		log.Printf("Error from UserUsecase.HandleCustomerMatchDecision for match %s: %v", req.MatchID, err)
		// Send a generic error or a more specific one if the usecase returns distinguishable errors
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInternalFailure, "Failed to process match decision.")
	}

	log.Printf("Successfully processed customer match decision for match %s via usecase.", req.MatchID)

	// Send a confirmation back to the client via WebSocket (optional, good practice)
	// The actual message content might vary based on whether the usecase call was truly successful
	// in terms of NATS publishing, which is abstracted away here.
	return m.manager.SendMessage(client.Conn, "server.customer.match_response_ack", map[string]string{"matchID": req.MatchID, "status": "processed_by_usecase"})
}

// NOTE: The new handler 'handleCustomerMatchResponse' needs to be registered in the
// WebSocketManager's message routing logic (e.g., in a switch statement or map
// that calls handlers based on the WebSocket message type, like "client.customer.match_response").
// This registration is typically in a file like 'manager.go' or a central router.

// handleRideArrived processes ride arrival events from WebSocket clients
func (m *WebSocketManager) handleRideArrived(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideCompleteEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride arrival format")
	}

	// Publish to NATS for rides-service to process completion
	m.userUC.RideArrived(context.Background(), &event)

	return nil
}
