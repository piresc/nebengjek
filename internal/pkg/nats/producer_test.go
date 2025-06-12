package nats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewProducer tests the creation of a new NATS producer
func TestNewProducer(t *testing.T) {
	t.Run("NewProducer with invalid address", func(t *testing.T) {
		// Test with an invalid NATS server address
		producer, err := NewProducer("invalid://address")
		assert.Error(t, err)
		assert.Nil(t, producer)
		assert.Contains(t, err.Error(), "failed to connect to NATS server")
	})

	t.Run("NewProducer with empty address", func(t *testing.T) {
		// Test with empty address
		producer, err := NewProducer("")
		assert.Error(t, err)
		assert.Nil(t, producer)
	})
}