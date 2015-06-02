// The channel handler provides an unbuffered `chan *event.Event`
// onto which each recieved event is written. Events are read from the chan
// using `Next()`.
//
// NB: since the `chan *event.Event` is unbuffered, `Event()` will block until some other
// goroutine reads from the chan.
package channel

import (
	"time"

	"github.com/phemmer/sawmill/event"
)

// Handler sends log events to an unbuffered chan *event.Event. Events are read with Next().
type Handler struct {
	channel chan *event.Event
}

// NewHandler creates and returns a new Handler
func NewHandler() *Handler {
	return &Handler{
		channel: make(chan *event.Event),
	}
}

// Event fills the sawmill.Handler interface for logging events
func (handler *Handler) Event(logEvent *event.Event) error {
	handler.channel <- logEvent
	return nil
}

// Next returns the next event by reading from channel, with an optional timeout. If timeout
// is 0, then Next performs a non-blocking read.
func (handler *Handler) Next(timeout time.Duration) *event.Event {
	var logEvent *event.Event
	if timeout == 0 {
		select {
		case logEvent = <-handler.channel:
		default:
		}
	} else {
		select {
		case logEvent = <-handler.channel:
		case <-time.After(timeout):
		}
	}
	return logEvent
}
