// The capture handler maintains a threadsafe slice of *event.Event. Events are added to the slice
// in the order that they are recieved. Provided for convenience:
//
//  * `Last()` func which returns the last event in the slice
//  * `Events()` func which returns a copy of the events slice
//  * `Clear()` func which drops all events in the slice
package capture

import (
	"sync"

	"github.com/phemmer/sawmill/event"
)

// Handler captures events in a threadsafe slice.
type Handler struct {
	events []*event.Event
	mutex  sync.Mutex
}

// NewHandler creates and returns a new Handler
func NewHandler() *Handler {
	return &Handler{
		events: []*event.Event{},
	}
}

// Event fills the sawmill.Handler interface
func (handler *Handler) Event(logEvent *event.Event) error {
	handler.mutex.Lock()
	handler.events = append(handler.events, logEvent)
	handler.mutex.Unlock()
	return nil
}

// Last returns the last event captured
func (handler *Handler) Last() *event.Event {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()

	l := len(handler.events)
	if l == 0 {
		return nil
	}

	logEvent := handler.events[l-1]
	return logEvent
}

// Events returns a slice of captured events
func (handler *Handler) Events() []*event.Event {
	handler.mutex.Lock()
	dst := make([]*event.Event, len(handler.events))
	_ = copy(dst, handler.events)
	handler.mutex.Unlock()

	return dst
}

// Clear drops all captured events
func (handler *Handler) Clear() {
	handler.mutex.Lock()
	handler.events = []*event.Event{}
	handler.mutex.Unlock()
}
