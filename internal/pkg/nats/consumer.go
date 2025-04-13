package nats

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// MessageHandler is a function that processes NATS messages
type MessageHandler func(message []byte) error

// Consumer handles consuming messages from NATS topics
type Consumer struct {
	conn         *nats.Conn
	subscription *nats.Subscription
}

// NewConsumer creates a new NATS consumer for a topic/channel
func NewConsumer(topic, queueGroup, address string, handler MessageHandler) (*Consumer, error) {
	// Connect to NATS server
	conn, err := nats.Connect(address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	// Subscribe to the topic with optional queue group
	var subscription *nats.Subscription
	if queueGroup != "" {
		subscription, err = conn.QueueSubscribe(topic, queueGroup, func(msg *nats.Msg) {
			// Process the message
			err := handler(msg.Data)
			if err != nil {
				log.Printf("Error processing message: %v", err)
			}
		})
	} else {
		subscription, err = conn.Subscribe(topic, func(msg *nats.Msg) {
			// Process the message
			err := handler(msg.Data)
			if err != nil {
				log.Printf("Error processing message: %v", err)
			}
		})
	}

	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	return &Consumer{conn: conn, subscription: subscription}, nil
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	if c.subscription != nil {
		c.subscription.Unsubscribe()
	}
	c.conn.Close()
}
