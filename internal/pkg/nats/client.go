package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

// Client represents a NATS client for publishing and subscribing to messages
type Client struct {
	conn *nats.Conn
}

// NewClient creates a new NATS client
func NewClient(url string) (*Client, error) {
	// Connect to NATS server
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	return &Client{conn: conn}, nil
}

// GetDB returns the underlying sqlx DB instance
func (c *Client) GetConn() *nats.Conn {
	return c.conn
}

// Publish sends a message to the specified subject
func (c *Client) Publish(subject string, data []byte) error {
	err := c.conn.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Subscribe subscribes to a subject and returns a subscription
func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject: %w", err)
	}

	return sub, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
