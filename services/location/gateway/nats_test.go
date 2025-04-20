package gateway

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	natsserver "github.com/nats-io/nats-server/test"
)

var (
	testNatsServer *server.Server
	testNatsURL    = "nats://127.0.0.1:8369"
)

func TestMain(m *testing.M) {
	testNatsServer = RunServerOnPort(8369)
	code := m.Run()
	testNatsServer.Shutdown()
	os.Exit(code)
}

func RunServerOnPort(port int) *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.Port = port
	return RunServerWithOptions(&opts)
}

func RunServerWithOptions(opts *server.Options) *server.Server {
	return natsserver.RunServer(opts)
}

func setupNatsConn(t *testing.T) *nats.Conn {
	nc, err := nats.Connect(testNatsURL)
	require.NoError(t, err, "Failed to connect to NATS server")
	return nc
}

// TestPublishLocationAggregate_Success tests successful publishing of location aggregates
func TestPublishLocationAggregate_Success(t *testing.T) {
	// Create mock NATS client
	nc := setupNatsConn(t)
	defer nc.Close()
	// Create test data
	aggregate := models.LocationAggregate{
		RideID:    "ride-123",
		Distance:  0.5,
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	// Channel to receive the message
	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.QueueSubscribe(constants.SubjectLocationAggregate, "test-subject", func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	locationGW := NewLocationGW(nc)
	err = locationGW.PublishLocationAggregate(context.Background(), aggregate)

	// Wait for the message
	select {
	case msg := <-msgCh:
		var publishedAggregate models.LocationAggregate
		err = json.Unmarshal(msg.Data, &publishedAggregate)
		require.NoError(t, err)

		assert.Equal(t, aggregate.RideID, publishedAggregate.RideID)
		assert.Equal(t, aggregate.Distance, publishedAggregate.Distance)
		assert.Equal(t, aggregate.Latitude, publishedAggregate.Latitude)
		assert.Equal(t, aggregate.Longitude, publishedAggregate.Longitude)
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive published message")
	}
}
