package sawmill

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/handler/channel"
	"github.com/phemmer/sawmill/handler/filter"
	"github.com/stretchr/testify/assert"
)

func TestLoggerEvent(t *testing.T) {
	handler := channel.NewHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler)
	defer logger.RemoveHandler("TestEvent", false)

	logger.Event(InfoLevel, "TestEvent", nil)

	logEvent := handler.Next(time.Second)

	if assert.NotNil(t, logEvent) {
		assert.Equal(t, logEvent.Message, "TestEvent")
	}
}

// this makes sure that a removed handler processes no further events
func TestLoggerRemoveHandler(t *testing.T) {
	handler := channel.NewHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler)
	logger.RemoveHandler("TestEvent", true)

	logger.Sync(logger.Event(InfoLevel, "TestEvent"))

	assert.Nil(t, handler.Next(time.Millisecond))
}

// check that removing a handler already removed doesn't error
func TestLoggerRemoveHandlerTwice(t *testing.T) {
	handler := channel.NewHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler)
	logger.RemoveHandler("TestEvent", false)
	logger.RemoveHandler("TestEvent", false)
}

// check that removing a handler waits for the handler to finish processing
func TestLoggerRemoveHandlerWait(t *testing.T) {
	handler := channel.NewHandler()

	logger := NewLogger()
	logger.AddHandler("TestEvent", handler)

	eventId1 := logger.Event(InfoLevel, "TestEvent")

	// first confirm the event is sitting unprocessed
	assert.Equal(t, logger.eventHandlerMap["TestEvent"].lastSentEventId, eventId1)
	assert.NotEqual(t, logger.eventHandlerMap["TestEvent"].lastProcessedEventId, eventId1)

	// send a second event, just so we have one that's not sitting on the channel.Handler channel
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

	handler1 := channel.NewHandler()
	logger.AddHandler("TestEvent", handler1)
	defer logger.RemoveHandler("TestEvent", false)

	handler2 := channel.NewHandler()
	logger.AddHandler("TestEvent", handler2)

	logger.Event(InfoLevel, "TestEvent")

	assert.Nil(t, handler1.Next(time.Millisecond))
	assert.NotNil(t, handler2.Next(time.Millisecond))
}

func TestLoggerFilterHandler(t *testing.T) {
	logger := NewLogger()

	channelHandler := channel.NewHandler()

	handler := logger.FilterHandler(channelHandler)
	assert.IsType(t, &filter.FilterHandler{}, handler)
}

func TestLoggerHelpers(t *testing.T) {
	logger := NewLogger()
	defer logger.Stop()

	handler := channel.NewHandler()
	logger.AddHandler("TestEvent", handler)

	type testLoggerHelper struct {
		String string
		Func   func(string, ...interface{}) uint64
		Level  event.Level
	}
	testHelpers := []testLoggerHelper{
		{"Emergency", logger.Emergency, EmergencyLevel},
		{"Alert", logger.Alert, AlertLevel},
		{"Critical", logger.Critical, CriticalLevel},
		{"Error", logger.Error, ErrorLevel},
		{"Warning", logger.Warning, WarningLevel},
		{"Notice", logger.Notice, NoticeLevel},
		{"Info", logger.Info, InfoLevel},
		{"Debug", logger.Debug, DebugLevel},
	}
	for _, helper := range testHelpers {
		message := fmt.Sprintf("TestHelper %s", helper.String)

		helper.Func(message, Fields{"helper": helper.String})

		logEvent := handler.Next(time.Millisecond)

		if assert.NotNil(t, logEvent) {
			assert.Equal(t, logEvent.Message, message)

			assert.Equal(t, logEvent.Level, helper.Level)

			if assert.NotNil(t, logEvent.Fields) {
				assert.Equal(t, logEvent.FlatFields["helper"], helper.String)
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

	handler := channel.NewHandler()
	logger.AddHandler("TestEvent", handler)

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

func TestLoggerCheckPanic(t *testing.T) {
	var panicErr interface{}
	defer func() {
		panicErr = recover()
	}()

	logger := NewLogger()
	handler := channel.NewHandler()
	logger.AddHandler("TestLoggerCheckPanic", handler)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// goroutine is because the channel.Handler is unbuffered, meaning the
		// CheckPanic will block trying to write a log event until handler.Next() is
		// called.
		defer func() {
			panicErr = recover()
			wg.Done()
		}()
		defer logger.CheckPanic()
		panic("this is a panic")
	}()

	logEvent := handler.Next(time.Second)

	assert.Equal(t, "panic", logEvent.Message)
	assert.Equal(t, "this is a panic", logEvent.FlatFields["error"])

	// make sure the panic was passed through
	wg.Wait()
	assert.Equal(t, "this is a panic", panicErr)
}

// this test is a bit complicated because we have to deal with synchronizing 2 goroutines
// The idea is:
// * We generate an event
// * A goroutine calls logger.Sync() to wait for the handler to process it
// * The handler doesn't process it until we call handler.Next()
// * handler.Next() gives us the event which we store in a shared variable
// * The Sync() goroutine notifies the main goroutine that it has set the shared variable
// * The main goroutine checks that the shared variable matches the generated event
// Then we run it 100 times just to make sure there's no racyness (there shouldn't be, but we want to make sure)
func TestSync(t *testing.T) {
	logger := NewLogger()
	defer logger.Stop()

	handler := channel.NewHandler()
	logger.AddHandler("TestEvent", handler)

	var lastHandledEventId uint64
	cond := sync.NewCond(&sync.Mutex{})
	for i := 0; i < 100; i++ {

		eventId := logger.Info("Test sync", Fields{"i": i})
		go func() {
			logger.Sync(eventId) // will block waiting for handler.Next() to be called
			cond.L.Lock()
			lastHandledEventId = eventId
			cond.Broadcast()
			cond.L.Unlock()
		}()
		runtime.Gosched()

		// This is the main bit of possible racyness, which is the reason for runtime.Gosched() above
		// It shouldn't be racy at all, but in case it is, try to catch it
		cond.L.Lock()
		assert.True(t, lastHandledEventId < eventId)
		cond.L.Unlock()

		cond.L.Lock()
		logEvent := handler.Next(time.Millisecond) // this should cause the logger.Sync() to return

		cond.Wait()
		assert.Equal(t, lastHandledEventId, eventId)
		assert.Equal(t, eventId, logEvent.Id)
		cond.L.Unlock()
	}
}

func TestSetGetSync(t *testing.T) {
	logger := NewLogger()
	defer logger.Stop()

	logger.SetSync(true)
	assert.True(t, logger.GetSync())

	logger.SetSync(false)
	assert.False(t, logger.GetSync())
}

func TestSyncMode(t *testing.T) {
	logger := NewLogger()
	defer logger.Stop()

	handler := channel.NewHandler()
	logger.AddHandler("TestEvent", handler)
	logger.SetSync(true)

	//TODO might be nicer to have something where we get a stack trace, start the
	// operation, get another stack trace, find the new goroutine, and verify it's
	// within a logger.Sync() call.
	for i := 0; i < 100; i++ {
		cond := sync.NewCond(&sync.Mutex{})
		complete := false
		go func() {
			logger.Info("Test sync", Fields{"i": i})
			cond.L.Lock()
			complete = true
			cond.Broadcast()
			cond.L.Unlock()
		}()

		runtime.Gosched()

		cond.L.Lock()
		assert.False(t, complete)
		cond.L.Unlock()

		// unblock the .Info() call
		cond.L.Lock()
		handler.Next(time.Millisecond)
		cond.Wait()
		assert.True(t, complete)
		cond.L.Unlock()
	}
}

func TestSetStackMinLevel(t *testing.T) {
	logger := NewLogger()
	defer logger.Stop()

	handler := channel.NewHandler()
	logger.AddHandler("TestSetStackMinLevel", handler)
	logger.SetStackMinLevel(ErrorLevel)

	logger.Warning("foo")
	logEvent := handler.Next(time.Second)
	assert.Empty(t, logEvent.Stack)

	logger.Error("bar")
	logEvent = handler.Next(time.Second)
	assert.NotEmpty(t, logEvent.Stack)
	// Unfortunately we can't (easily) test that the stack trace includes this
	// test function as event.New looks for the first function outside
	// the sawmill package, and we're testing inside the package.
	// However it should at least include the testing package as that's what
	// called us.
}

func TestGetStackMinLevel(t *testing.T) {
	logger := NewLogger()
	defer logger.Stop()

	assert.True(t, logger.GetStackMinLevel() > EmergencyLevel)

	logger.SetStackMinLevel(DebugLevel)
	assert.Equal(t, DebugLevel, logger.GetStackMinLevel())
}

// Test dropping
// Test Stop()
