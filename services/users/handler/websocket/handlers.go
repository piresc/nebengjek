package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8" // Import redis package
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/converter" // Kept for other handlers if they use it
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// ClientCustomerMatchResponse defines the structure for the 'client.customer.match_response' WebSocket message.
// DriverID is removed as it will be fetched from the server-side cache.
type ClientCustomerMatchResponse struct {
	MatchID   string `json:"matchID"`
	Confirmed bool   `json:"confirmed"`
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
// This handler is for the driver's initial acceptance of a match found by the system.
func (m *WebSocketManager) handleMatchAccept(client *models.WebSocketClient, data json.RawMessage) error {
	UserID := client.UserID // This is the DriverID

	var matchProposalFromDriver models.MatchProposal
	if err := json.Unmarshal(data, &matchProposalFromDriver); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid match proposal format")
	}

	// The User Service use case `ConfirmMatch` is expected to handle the logic for a driver's initial acceptance,
	// which includes setting the status to MatchStatusPendingCustomerConfirmation and publishing the
	// NATS event (e.g., constants.SubjectMatchAccepted) that the MatchService will consume.
	log.Printf("handleMatchAccept: Driver %s responding to match %s. Payload: %+v", UserID, matchProposalFromDriver.ID, matchProposalFromDriver)

	err := m.userUC.ConfirmMatch(context.Background(), &matchProposalFromDriver, UserID)
	if err != nil {
		log.Printf("Error confirming match for driver %s (matchID: %s): %v", client.UserID, matchProposalFromDriver.ID, err)
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
		wsClient, exists := m.manager.GetClient(driverID) // Use m.manager.GetClient
		if !exists {
			log.Printf("Client with ID %s not found for sending location update error", driverID)
			// If client not found, we can't send WS error, so just return the error.
			return fmt.Errorf("client %s not found, cannot send error for location update: %w", driverID, err)
		}
		log.Printf("Error updating location for user %s: %v", driverID, err)
		return m.manager.SendErrorMessage(wsClient.Conn, constants.ErrorInvalidLocation, err.Error())
	}

	return nil
}

// handleCustomerMatchResponse processes the customer's response (acceptance/rejection) to a match confirmation request.
func (m *WebSocketManager) handleCustomerMatchResponse(client *models.WebSocketClient, data json.RawMessage) error {
	ctx := context.Background()
	var req ClientCustomerMatchResponse // Contains MatchID and Confirmed

	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("handleCustomerMatchResponse: Error unmarshalling customer response from user %s: %v", client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid customer match response format")
	}

	log.Printf("handleCustomerMatchResponse: Received response from UserID: %s, MatchID: %s, Confirmed: %t",
		client.UserID, req.MatchID, req.Confirmed)

	// Ensure Redis client is available
	if m.redisClient == nil {
		log.Printf("handleCustomerMatchResponse: CRITICAL - Redis client not available in WebSocketManager for match %s.", req.MatchID)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInternalFailure, "Service internal error, please try again.")
	}

	// Fetch MatchProposal from cache
	cacheKey := fmt.Sprintf("matchproposal:%s", req.MatchID)
	cachedData, err := m.redisClient.Get(ctx, cacheKey).Result()

	if err == redis.Nil {
		log.Printf("handleCustomerMatchResponse: Cached MatchProposal not found for MatchID %s (key: %s). UserID: %s", req.MatchID, cacheKey, client.UserID)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorMatchUpdateFailed, "Match details expired or not found. Please await a new match.")
	} else if err != nil {
		log.Printf("handleCustomerMatchResponse: Failed to retrieve MatchProposal from cache for matchID %s (key: %s), UserID %s: %v", req.MatchID, cacheKey, client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInternalFailure, "Error retrieving match details. Please try again.")
	}

	var cachedProposal models.MatchProposal
	if err := json.Unmarshal([]byte(cachedData), &cachedProposal); err != nil {
		log.Printf("handleCustomerMatchResponse: Failed to unmarshal cached MatchProposal for matchID %s (key: %s), UserID %s: %v", req.MatchID, cacheKey, client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInternalFailure, "Error processing match details. Please try again.")
	}

	// CRITICAL VALIDATION: Ensure the user responding is the correct passenger for this match.
	if cachedProposal.PassengerID != client.UserID {
		log.Printf("handleCustomerMatchResponse: CRITICAL - UserID mismatch. Client %s, Cached PassengerID %s for MatchID %s.",
			client.UserID, cachedProposal.PassengerID, req.MatchID)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorMatchUpdateFailed, "User authentication error for this match.")
	}

	// Use the authoritative cached data, only updating status based on client's confirmation.
	matchProposalToSend := cachedProposal // This now contains DriverID, locations etc. from the cache.
	matchProposalToSend.ID = req.MatchID  // Ensure MatchID from request is used (should be same as cachedProposal.ID)

	var natsSubject string
	if req.Confirmed {
		matchProposalToSend.MatchStatus = models.MatchStatusAccepted
		natsSubject = constants.SubjectCustomerMatchConfirmed
	} else {
		matchProposalToSend.MatchStatus = models.MatchStatusRejected
		natsSubject = constants.SubjectCustomerMatchRejected
	}

	// Delegate to the user usecase.
	log.Printf("Calling UserUsecase.HandleCustomerMatchDecision for match %s, UserID %s, Subject: %s, Status: %s.",
		req.MatchID, client.UserID, natsSubject, matchProposalToSend.MatchStatus)

	if err := m.userUC.HandleCustomerMatchDecision(ctx, matchProposalToSend, natsSubject); err != nil {
		log.Printf("Error from UserUsecase.HandleCustomerMatchDecision for match %s, UserID %s: %v", req.MatchID, client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInternalFailure, "Failed to process match decision.")
	}

	log.Printf("Successfully processed customer match decision for match %s, UserID %s.", req.MatchID, client.UserID)

	// Delete the cache entry after successful processing to prevent reuse.
	if errDel := m.redisClient.Del(ctx, cacheKey).Err(); errDel != nil {
		log.Printf("handleCustomerMatchResponse: INFO - Failed to delete MatchProposal from cache for key %s after processing (UserID: %s): %v", cacheKey, client.UserID, errDel)
		// Do not fail the operation if cache deletion fails, as primary operation succeeded.
	} else {
		log.Printf("handleCustomerMatchResponse: Successfully deleted MatchProposal from cache for key %s (UserID: %s).", cacheKey, client.UserID)
	}

	return m.manager.SendMessage(client.Conn, "server.customer.match_response_ack", map[string]string{"matchID": req.MatchID, "status": "processed"})
}

// NOTE: The new handler 'handleCustomerMatchResponse' needs to be registered in the
// WebSocketManager's message routing logic (e.g., in a switch statement or map
// that calls handlers based on the WebSocket message type, like "client.customer.match_response").
// This registration is typically in a file like 'manager.go' or a central router.

// handleRideArrived processes ride arrival events from WebSocket clients
func (m *WebSocketManager) handleRideArrived(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideCompleteEvent
	if err := json.Unmarshal(data, &req); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride arrival format")
	}

	// Publish to NATS for rides-service to process completion
	m.userUC.RideArrived(context.Background(), &event)

	return nil
}
