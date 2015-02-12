package sawmill

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/phemmer/sawmill/event"
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

// check that adding a handler under the same name overrides the first
func TestLoggerAddDuplicateHandler(t *testing.T) {
	logger := NewLogger()

	handler1 := NewChannelHandler()
	logger.AddHandler("TestEvent", handler1, DebugLevel, EmergencyLevel)
	defer logger.RemoveHandler("TestEvent", false)

	handler2 := NewChannelHandler()
	logger.AddHandler("TestEvent", handler2, DebugLevel, EmergencyLevel)

	logger.Event(InfoLevel, "TestEvent")

	assert.Nil(t, handler1.Next(time.Millisecond))
	assert.NotNil(t, handler2.Next(time.Millisecond))
}

func TestLoggerLevels(t *testing.T) {
	logger := NewLogger()
	defer logger.Stop()

	handler := NewChannelHandler()
	logger.AddHandler("TestEvent", handler, DebugLevel, EmergencyLevel)

	type testLoggerLevel struct {
		String string
		Func   func(string, ...interface{}) uint64
	}
	testLevels := []testLoggerLevel{
		{"Emergency", logger.Emergency},
		{"Alert", logger.Alert},
		{"Critical", logger.Critical},
		{"Error", logger.Error},
		{"Warning", logger.Warning},
		{"Notice", logger.Notice},
		{"Info", logger.Info},
		{"Debug", logger.Debug},
	}
	for _, level := range testLevels {
		message := fmt.Sprintf("TestLevel %s", level.String)

		level.Func(message, Fields{"level": level.String})

		logEvent := handler.Next(time.Millisecond)

		if assert.NotNil(t, logEvent) {
			assert.Equal(t, logEvent.Message, message)

			if assert.NotNil(t, logEvent.Fields) {
				assert.Equal(t, logEvent.FlatFields["level"], level.String)
			}
		}
	}
}

func TestLoggerFatal(t *testing.T) {
	exitBkup := exit
	var exitCode int
	exit = func(code int) { exitCode = code }
	defer func() { exit = exitBkup }()

	logger := NewLogger()
	defer logger.Stop()

	handler := NewChannelHandler()
	logger.AddHandler("TestEvent", handler, DebugLevel, EmergencyLevel)

	// logger.Fatal performs a Stop(), which waits for the handler to process the event. So we have to process it or we deadlock. We do this by starting a goroutine
	var logEvent *event.Event
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		logEvent = handler.Next(time.Second)
		wg.Done()
	}()

	logger.Fatal("TestHelper Fatal", Fields{"helper": "Fatal"})
	wg.Wait()

	if assert.NotNil(t, logEvent) {
		assert.Equal(t, logEvent.Message, "TestHelper Fatal")

		if assert.NotNil(t, logEvent.Fields) {
			assert.Equal(t, logEvent.FlatFields["helper"], "Fatal")
		}
	}

	assert.Equal(t, exitCode, 1)
}

// Test Sync()
// Test Stop()
