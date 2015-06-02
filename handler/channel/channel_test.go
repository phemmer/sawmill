package channel

import (
	"runtime"
	"testing"
	"time"

	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/event"
	"github.com/stretchr/testify/assert"
)

func TestHandlerIface(t *testing.T) {
	assert.Implements(t, (*sawmill.Handler)(nil), NewHandler())
}

func TestEvent(t *testing.T) {
	ch := NewHandler()
	logEvent := makeEvent(0)
	go ch.Event(logEvent)

	var chEvent *event.Event
	select {
	case chEvent = <-ch.channel:
	case <-time.After(time.Millisecond):
	}

	assert.Equal(t, logEvent, chEvent)
}

func TestNext(t *testing.T) {
	ch := NewHandler()
	logEvent := makeEvent(0)
	go func() { ch.channel <- logEvent }()

	chEvent := ch.Next(time.Millisecond)

	assert.Equal(t, logEvent, chEvent)
}

func TestNextTimeout(t *testing.T) {
	ch := NewHandler()

	chEvent := ch.Next(time.Nanosecond)

	assert.Nil(t, chEvent)
}

func TestNextNonblocking(t *testing.T) {
	ch := NewHandler()
	logEvent := makeEvent(0)
	go func() {
		time.Sleep(10 * time.Millisecond)
		ch.channel <- logEvent
	}()

	chEvent := ch.Next(time.Duration(0))

	assert.Nil(t, chEvent)
}

var eventCounter uint64

func makeEvent(level event.Level) *event.Event {
	eventCounter++

	callerPC, _, _, _ := runtime.Caller(1)
	callerFunc := runtime.FuncForPC(callerPC)
	callerName := callerFunc.Name()

	message := "testing " + callerName + "()"
	data := map[string]interface{}{"test": callerName}

	return event.New(eventCounter, level, message, data, false)
}
