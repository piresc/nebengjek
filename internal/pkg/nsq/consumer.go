package nsq

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nsqio/go-nsq"
)

// MessageHandler is a function that processes NSQ messages
type MessageHandler func(message []byte) error

// Consumer handles consuming messages from NSQ topics
type Consumer struct {
	consumer *nsq.Consumer
}

// NewConsumer creates a new NSQ consumer for a topic/channel
func NewConsumer(topic, channel, address string, handler MessageHandler) (*Consumer, error) {
	config := nsq.NewConfig()

	// Create a new consumer for the specified topic/channel
	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQ consumer: %w", err)
	}

	// Set the message handler
	consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		// Mark the message as processed
		message.Touch()

		// Process the message
		err := handler(message.Body)
		if err != nil {
			log.Printf("Error processing message: %v", err)
			// Requeue the message for later processing
			return err
		}

		// Mark the message as finished
		message.Finish()
		return nil
	}))

	// Connect to the NSQ daemon
	err = consumer.ConnectToNSQD(address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NSQ daemon: %w", err)
	}

	return &Consumer{consumer: consumer}, nil
}

// ConnectToLookupd connects the consumer to NSQ lookupd instances
func (c *Consumer) ConnectToLookupd(addresses []string) error {
	for _, addr := range addresses {
		err := c.consumer.ConnectToNSQLookupd(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to NSQ lookupd at %s: %w", addr, err)
		}
	}
	return nil
}

// UnmarshalMessage deserializes a JSON message into the provided struct
func UnmarshalMessage(messageBody []byte, v interface{}) error {
	err := json.Unmarshal(messageBody, v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return nil
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	c.consumer.Stop()
}
