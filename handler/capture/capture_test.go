package capture

import (
	"runtime"
	"testing"

	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/event"
	"github.com/stretchr/testify/assert"
)

func TestHandlerIface(t *testing.T) {
	assert.Implements(t, (*sawmill.Handler)(nil), NewHandler())
}

func TestEvent(t *testing.T) {
	ch := NewHandler()
	logEvent0 := makeEvent(0)
	logEvent1 := makeEvent(1)
	logEvent2 := makeEvent(2)

	ch.Event(logEvent0)
	ch.Event(logEvent1)
	ch.Event(logEvent2)

	assert.Equal(t, logEvent0, ch.events[0])
	assert.Equal(t, logEvent1, ch.events[1])
	assert.Equal(t, logEvent2, ch.events[2])
}

func TestLast(t *testing.T) {
	ch := NewHandler()
	logEvent2 := makeEvent(2)

	ch.Event(makeEvent(0))
	ch.Event(makeEvent(1))
	ch.Event(logEvent2)

	assert.Equal(t, logEvent2, ch.Last())
}

func TestLastEmpty(t *testing.T) {
	ch := NewHandler()

	assert.Nil(t, ch.Last())
}

func TestEvents(t *testing.T) {
	ch := NewHandler()
	events := []*event.Event{
		makeEvent(0),
		makeEvent(1),
		makeEvent(2),
	}

	for _, e := range events {
		ch.Event(e)
	}

	assert.Equal(t, events, ch.Events())
}

func TestClear(t *testing.T) {
	ch := NewHandler()
	for i := 0; i < 5; i++ {
		ch.Event(makeEvent(0))
	}
	assert.Len(t, ch.events, 5)

	ch.Clear()

	assert.Len(t, ch.events, 0)
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
