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

	assert.Nil(t, handler.Next(time.Millisecond))
}

// check that removing a handler already removed doesn't error
func TestLoggerRemoveHandlerTwice(t *testing.T) {
	handler := NewChannelHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler, DebugLevel, EmergencyLevel)
	logger.RemoveHandler("TestEvent", false)
	logger.RemoveHandler("TestEvent", false)
}

// check that removing a handler waits for the handler to finish processing
func TestLoggerRemoveHandlerWait(t *testing.T) {
	handler := NewChannelHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler, DebugLevel, EmergencyLevel)

	eventId1 := logger.Event(InfoLevel, "TestEvent")

	// first confirm the event is sitting unprocessed
	assert.Equal(t, logger.eventHandlerMap["TestEvent"].lastSentEventId, eventId1)
	assert.NotEqual(t, logger.eventHandlerMap["TestEvent"].lastProcessedEventId, eventId1)

	// send a second event, just so we have one that's not sitting on the channelHandler channel
	eventId2 := logger.Event(InfoLevel, "TestEvent")
	assert.Equal(t, logger.eventHandlerMap["TestEvent"].lastSentEventId, eventId2)
	assert.NotEqual(t, logger.eventHandlerMap["TestEvent"].lastProcessedEventId, eventId2)

	logger.RemoveHandler("TestEvent", false)

	event1 := handler.Next(time.Second)
	assert.NotNil(t, event1)
	assert.Equal(t, event1.Id, eventId1)

	event2 := handler.Next(time.Second)
	assert.NotNil(t, event2.Id, eventId2)

	assert.Nil(t, handler.Next(time.Millisecond))
}
// Test all the helper functions
// Test Sync()
// Test Stop()
