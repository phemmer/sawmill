package sawmill

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoggerEvent(t *testing.T) {
	handler := NewChannelHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler, DebugLevel, EmergencyLevel)
	defer logger.RemoveHandler("TestEvent", false)

	logger.Event(InfoLevel, "TestEvent", nil)

	logEvent := handler.Next(time.Second)

	if assert.NotNil(t, logEvent) {
		assert.Equal(t, logEvent.Message, "TestEvent")
	}
}

// this makes sure that a removed handler processes no further events
func TestLoggerRemoveHandler(t *testing.T) {
	handler := NewChannelHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler, DebugLevel, EmergencyLevel)
	logger.RemoveHandler("TestEvent", true)

	logger.Sync(logger.Event(InfoLevel, "TestEvent"))

	assert.Nil(t, handler.Next(0))
}

// Test that removing a handler already removed doesn't error
// Test that removing a handler waits for the handler to finish processing
// Test all the helper functions
// Test Sync()
// Test Stop()
