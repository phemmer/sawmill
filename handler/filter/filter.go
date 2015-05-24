/*
The filter package provides a way to filter events from handlers.

The filter itself is just a handler which sits in front of another handler. When the filter handler receives an event, it iterates through all its rules (filter functions), and if they all pass, the event is relayed to the next handler in the chain.
*/
package filter

import (
	"reflect"

	"github.com/phemmer/sawmill/event"
)

// Handler represents a destination for sawmill to send events to.
//
// This is copied from the base sawmill package. Sawmill is not imported so that the base package can import the filter package without a dependency cycle.
type Handler interface {
	Event(event *event.Event) error
}

// FilterFunc is the signature for a filter used by the handler.
// The function returns `true` to indicate the event should be allowed, and `false` to indicate it should be dropped.
type FilterFunc func(*event.Event) bool

type FilterHandler struct {
	nextHandler Handler
	filterFuncs []FilterFunc
}

// New creates a new FilterHandler which relays events to the handler specified in `nextHandler`.
//
// If any filterFuncs are provided, they are used as the initial filter list.
func New(nextHandler Handler, filterFuncs ...FilterFunc) *FilterHandler {
	return &FilterHandler{
		nextHandler: nextHandler,
		filterFuncs: filterFuncs,
	}
}

// Event processes an event through the filters, relaying the event to the next handler if all the filters pass.
func (filterHandler *FilterHandler) Event(logEvent *event.Event) error {
	for _, filterFunc := range filterHandler.filterFuncs {
		if !filterFunc(logEvent) {
			return nil
		}
	}
	return filterHandler.nextHandler.Event(logEvent)
}

// Filter adds a check function to the filter.
//
// The function is passed the event, and should return true if the event is allowed, and false otherwise.
//
// The return value is the handler itself. This is to allow chaining multiple operations together.
func (filterHandler *FilterHandler) Filter(filterFuncs ...FilterFunc) *FilterHandler {
	filterHandler.filterFuncs = append(filterHandler.filterFuncs, filterFuncs...)

	return filterHandler
}

// LevelMin adds a canned filter to the handler which rejects events with a level less than the one specified.
//
// The return value is the handler itself. This is to allow chaining multiple operations together.
func (filterHandler *FilterHandler) LevelMin(levelMin event.Level) *FilterHandler {
	filterFunc := func(logEvent *event.Event) bool {
		if logEvent.Level < levelMin {
			return false
		}
		return true
	}

	return filterHandler.Filter(filterFunc)
}

// LevelMax adds a canned filter to the handler which rejects events with a level greater than the one specified.
//
// The return value is the handler itself. This is to allow chaining multiple operations together.
func (filterHandler *FilterHandler) LevelMax(levelMax event.Level) *FilterHandler {
	filterFunc := func(logEvent *event.Event) bool {
		if logEvent.Level > levelMax {
			return false
		}
		return true
	}

	return filterHandler.Filter(filterFunc)
}

// Dedup adds a canned filter to the handler which suppresses duplicate
// messages.
//
// When an event is received that contains the same message & fields as a
// previous message, the message is not sent on to the next handler. Once a
// different message is received, the filter generates a summary message
// indicating how many duplicates were suppressed.
//
// The return value is the handler itself. This is to allow chaining multiple operations together.
func (filterHandler *FilterHandler) Dedup() *FilterHandler {
	var lastLogEvent *event.Event
	var dups int
	filterFunc := func(logEvent *event.Event) bool {
		if lastLogEvent == nil {
			lastLogEvent = logEvent
			return true
		}

		if lastLogEvent.Message == logEvent.Message && reflect.DeepEqual(lastLogEvent.FlatFields, logEvent.FlatFields) {
			dups++
			lastLogEvent = logEvent
			return false
		}

		if dups > 0 {
			dupEvent := event.New(
				lastLogEvent.Id,
				event.Notice,
				"duplicates of last log event suppressed",
				map[string]int{"count": dups},
				false,
			)
			filterHandler.nextHandler.Event(dupEvent)
		}

		dups = 0
		lastLogEvent = logEvent
		return true
	}

	return filterHandler.Filter(filterFunc)
}
