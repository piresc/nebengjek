package nats

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// Producer handles publishing messages to NATS topics
type Producer struct {
	conn *nats.Conn
}

// NewProducer creates a new NATS producer
func NewProducer(address string) (*Producer, error) {
	// Connect to NATS server
	conn, err := nats.Connect(address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	return &Producer{conn: conn}, nil
}

// Publish sends a message to the specified topic
func (p *Producer) Publish(topic string, message interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = p.conn.Publish(topic, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message to topic: %s", topic)
	return nil
}

// PublishAsync sends a message to the specified topic asynchronously
func (p *Producer) PublishAsync(topic string, message interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// NATS doesn't have a direct async publish like NSQ, but it's already non-blocking
	err = p.conn.Publish(topic, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Stop gracefully closes the NATS connection
func (p *Producer) Stop() {
	p.conn.Close()
}
