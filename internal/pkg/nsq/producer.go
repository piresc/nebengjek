package nsq

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nsqio/go-nsq"
)

// Producer handles publishing messages to NSQ topics
type Producer struct {
	producer *nsq.Producer
}

// NewProducer creates a new NSQ producer
func NewProducer(address string) (*Producer, error) {
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer(address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQ producer: %w", err)
	}

	// Ping the NSQ daemon to ensure connectivity
	err = producer.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping NSQ daemon: %w", err)
	}

	return &Producer{producer: producer}, nil
}

// Publish sends a message to the specified topic
func (p *Producer) Publish(topic string, message interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = p.producer.Publish(topic, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message to topic: %s", topic)
	return nil
}

// PublishAsync sends a message to the specified topic asynchronously
func (p *Producer) PublishAsync(topic string, message interface{}, doneChan chan *nsq.ProducerTransaction) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	p.producer.PublishAsync(topic, msgBytes, doneChan)
	return nil
}

// Stop gracefully stops the producer
func (p *Producer) Stop() {
	p.producer.Stop()
}
