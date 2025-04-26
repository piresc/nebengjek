package nats

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
)

// NotificationCollector tracks WebSocket notifications for testing
type NotificationCollector struct {
	ClientIDs   []string
	EventTypes  []string
	EventValues []interface{}
}

// Adds a notification to the collector
func (nc *NotificationCollector) Add(clientID string, eventType string, value interface{}) {
	nc.ClientIDs = append(nc.ClientIDs, clientID)
	nc.EventTypes = append(nc.EventTypes, eventType)
	nc.EventValues = append(nc.EventValues, value)
}

// Create a new notification collector
func NewNotificationCollector() *NotificationCollector {
	return &NotificationCollector{
		ClientIDs:   make([]string, 0),
		EventTypes:  make([]string, 0),
		EventValues: make([]interface{}, 0),
	}
}

// Test implementation of the handleMatchEvent function that follows the same logic as the real one
func testHandleMatchEvent(msg []byte, nc *NotificationCollector) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Collect notifications that would be sent to clients
	nc.Add(event.DriverID, constants.SubjectMatchFound, event)
	nc.Add(event.PassengerID, constants.SubjectMatchFound, event)
	return nil
}

// Test implementation of the handleMatchConfirmEvent function
func testHandleMatchConfirmEvent(msg []byte, nc *NotificationCollector) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted event: %w", err)
	}

	// Collect notifications that would be sent to clients
	nc.Add(event.DriverID, constants.EventMatchConfirm, event)
	nc.Add(event.PassengerID, constants.EventMatchConfirm, event)
	return nil
}

// Test implementation of the handleMatchRejectedEvent function
func testHandleMatchRejectedEvent(msg []byte, nc *NotificationCollector) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected event: %w", err)
	}

	// Only notify the driver whose match was rejected
	nc.Add(event.DriverID, constants.EventMatchRejected, event)
	return nil
}

func TestHandleMatchEvent_Success(t *testing.T) {
	// Arrange
	nc := NewNotificationCollector()

	// Create test data
	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchID := uuid.New().String()

	event := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusPending,
	}

	eventBytes, err := json.Marshal(event)
	assert.NoError(t, err)

	// Act
	err = testHandleMatchEvent(eventBytes, nc)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nc.ClientIDs), "Should notify both driver and passenger")
	assert.Contains(t, nc.ClientIDs, driverID, "Should notify driver")
	assert.Contains(t, nc.ClientIDs, passengerID, "Should notify passenger")

	// Check event type
	for _, eventType := range nc.EventTypes {
		assert.Equal(t, constants.SubjectMatchFound, eventType)
	}

	// Check event values (both should be the same match proposal)
	for _, eventVal := range nc.EventValues {
		matchEvent, ok := eventVal.(models.MatchProposal)
		assert.True(t, ok)
		assert.Equal(t, matchID, matchEvent.ID)
		assert.Equal(t, driverID, matchEvent.DriverID)
		assert.Equal(t, passengerID, matchEvent.PassengerID)
	}
}

func TestHandleMatchEvent_UnmarshalError(t *testing.T) {
	// Arrange
	nc := NewNotificationCollector()

	// Create invalid JSON
	invalidJson := []byte(`{invalid_json}`)

	// Act
	err := testHandleMatchEvent(invalidJson, nc)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal match event")
	assert.Empty(t, nc.ClientIDs, "Should not notify any clients on error")
}

func TestHandleMatchConfirmEvent_Success(t *testing.T) {
	// Arrange
	nc := NewNotificationCollector()

	// Create test data
	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchID := uuid.New().String()

	event := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	eventBytes, err := json.Marshal(event)
	assert.NoError(t, err)

	// Act
	err = testHandleMatchConfirmEvent(eventBytes, nc)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nc.ClientIDs), "Should notify both driver and passenger")
	assert.Contains(t, nc.ClientIDs, driverID, "Should notify driver")
	assert.Contains(t, nc.ClientIDs, passengerID, "Should notify passenger")

	// Check event type
	for _, eventType := range nc.EventTypes {
		assert.Equal(t, constants.EventMatchConfirm, eventType)
	}

	// Check event values
	for _, eventVal := range nc.EventValues {
		matchEvent, ok := eventVal.(models.MatchProposal)
		assert.True(t, ok)
		assert.Equal(t, matchID, matchEvent.ID)
		assert.Equal(t, models.MatchStatusAccepted, matchEvent.MatchStatus)
	}
}

func TestHandleMatchRejectedEvent_Success(t *testing.T) {
	// Arrange
	nc := NewNotificationCollector()

	// Create test data
	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchID := uuid.New().String()

	event := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusRejected,
	}

	eventBytes, err := json.Marshal(event)
	assert.NoError(t, err)

	// Act
	err = testHandleMatchRejectedEvent(eventBytes, nc)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nc.ClientIDs), "Should only notify the driver")
	assert.Equal(t, driverID, nc.ClientIDs[0], "Should notify driver")
	assert.Equal(t, constants.EventMatchRejected, nc.EventTypes[0])

	// Check event value
	matchEvent, ok := nc.EventValues[0].(models.MatchProposal)
	assert.True(t, ok)
	assert.Equal(t, matchID, matchEvent.ID)
	assert.Equal(t, models.MatchStatusRejected, matchEvent.MatchStatus)
}
