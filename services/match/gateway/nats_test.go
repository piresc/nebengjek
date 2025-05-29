package gateway

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	natsserver "github.com/nats-io/nats-server/test"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testNatsURL = "nats://127.0.0.1:8369"

func TestMain(m *testing.M) {
	opts := natsserver.DefaultTestOptions
	opts.Port = 8369
	testNatsServer := natsserver.RunServer(&opts)
	code := m.Run()
	testNatsServer.Shutdown()
	os.Exit(code)
}

// TestPublishMatchFound_Success tests successful publishing of match found events
func TestPublishMatchFound_Success(t *testing.T) {
	// Create NATS client
	nc, err := natspkg.NewClient(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	defer nc.Close()

	// Create test data
	matchProp := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusPending,
		DriverLocation: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		UserLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.Subscribe(constants.SubjectMatchFound, func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create gateway and publish message
	matchGW := NewMatchGW(nc)
	err = matchGW.PublishMatchFound(context.Background(), matchProp)
	require.NoError(t, err)

	// Wait for the message and verify contents
	select {
	case msg := <-msgCh:
		var receivedMatch models.MatchProposal
		err = json.Unmarshal(msg.Data, &receivedMatch)
		require.NoError(t, err)

		assert.Equal(t, matchProp.ID, receivedMatch.ID)
		assert.Equal(t, matchProp.DriverID, receivedMatch.DriverID)
		assert.Equal(t, matchProp.PassengerID, receivedMatch.PassengerID)
		assert.Equal(t, matchProp.MatchStatus, receivedMatch.MatchStatus)
		assert.Equal(t, matchProp.DriverLocation.Latitude, receivedMatch.DriverLocation.Latitude)
		assert.Equal(t, matchProp.DriverLocation.Longitude, receivedMatch.DriverLocation.Longitude)
		assert.Equal(t, matchProp.UserLocation.Latitude, receivedMatch.UserLocation.Latitude)
		assert.Equal(t, matchProp.UserLocation.Longitude, receivedMatch.UserLocation.Longitude)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}
