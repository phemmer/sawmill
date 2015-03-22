package sawmill

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/handler/syslog"
	"github.com/phemmer/sawmill/handler/writer"
)

var exit func(code int) = os.Exit // this is for testing so we can prevent an actual exit

// Fields is a convenience type for passing ancillary data when generating events.
type Fields map[string]interface{}

// Handler represents a destination for sawmill to send events to.
// It responds to a single method, `Event`, which accepts the event to process. It must not return until the event has been fully processed.
type Handler interface {
	Event(event *event.Event) error
}

type eventHandlerSpec struct {
	name          string
	eventChannel  chan *event.Event
	finishChannel chan bool

	lastSentEventId          uint64
	lastProcessedEventId     uint64
	lastProcessedEventIdCond *sync.Cond
}

// Logger is the core type in sawmill.
// The logger tracks a list of destinations, and when given an event, will asynchronously send that event to all registered destination handlers.
type Logger struct {
	eventHandlerMap map[string]*eventHandlerSpec
	mutex           sync.RWMutex
	waitgroup       sync.WaitGroup
	lastEventId     uint64
}

// NewLogger constructs a Logger.
// The new Logger will not have any registered handlers.
func NewLogger() *Logger {
	return &Logger{
		eventHandlerMap: make(map[string]*eventHandlerSpec),
	}
}

// AddHandler registers a new destination handler with the logger.
// The name parameter is a unique identifier so that the handler can be targeted with RemoveHandler().
// If a handler with the same name already exists, it will be replaced by the new one.
func (logger *Logger) AddHandler(name string, handler Handler) {
	//TODO check name collision
	spec := &eventHandlerSpec{
		name:                     name,
		eventChannel:             make(chan *event.Event, 100),
		finishChannel:            make(chan bool, 1),
		lastProcessedEventIdCond: sync.NewCond(&sync.Mutex{}),
	}

	logger.waitgroup.Add(1)
	go handlerDriver(spec, handler, &logger.waitgroup)

	logger.mutex.Lock()
	logger.eventHandlerMap[name] = spec
	logger.mutex.Unlock()
}
func handlerDriver(spec *eventHandlerSpec, handler Handler, waitgroup *sync.WaitGroup) {
	defer waitgroup.Done()

	eventChannel := spec.eventChannel
	finishChannel := spec.finishChannel

	for logEvent := range eventChannel {
		if logEvent == nil {
			break
		}

		handler.Event(logEvent) //TODO error handler

		spec.lastProcessedEventIdCond.L.Lock()
		spec.lastProcessedEventId = logEvent.Id
		spec.lastProcessedEventIdCond.Broadcast()
		spec.lastProcessedEventIdCond.L.Unlock()
	}

	finishChannel <- true
}

// RemoveHandler removes the named handler from the logger, preventing any further events from being sent to it.
// The wait parameter will result in the function blocking until all events queued for the handler have finished processing.
func (logger *Logger) RemoveHandler(name string, wait bool) {
	logger.mutex.Lock()
	eventHandlerSpec := logger.eventHandlerMap[name]
	if eventHandlerSpec == nil {
		// doesn't exist
		logger.mutex.Unlock()
		return
	}
	delete(logger.eventHandlerMap, name)
	logger.mutex.Unlock()
	eventHandlerSpec.eventChannel <- nil
	if !wait {
		return
	}
	<-eventHandlerSpec.finishChannel
}

// Stop removes all destination handlers on the logger, and waits for any pending events to flush out.
func (logger *Logger) Stop() {
	logger.mutex.RLock()
	handlerNames := make([]string, len(logger.eventHandlerMap))
	for handlerName, _ := range logger.eventHandlerMap {
		handlerNames = append(handlerNames, handlerName)
	}
	logger.mutex.RUnlock()

	for _, handlerName := range handlerNames {
		logger.RemoveHandler(handlerName, false)
	}

	logger.waitgroup.Wait() //TODO timeout?
}

// InitStdStreams is a convience function to register a STDOUT/STDERR handler with the logger.
//
// The handler is added with the name 'stdStreams'
func (logger *Logger) InitStdStreams() {
	logger.AddHandler("stdStreams", writer.NewStandardStreamsWriter())
}

// InitStdSyslog is a convenience function to register a syslog handler with the logger.
//
// The handler is added with the name 'syslog'
func (logger *Logger) InitStdSyslog() error {
	syslogHandler, err := syslog.NewSyslogWriter("", "", 0, "")
	if err != nil {
		return err
	}
	logger.AddHandler("syslog", syslogHandler)

	return nil
}

// Event queues a message at the given level.
// Additional fields may be provided, which will be recursively copied at the time of the function call, and provided to the destination output handler.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Event(level event.Level, message string, fields ...interface{}) uint64 {
	var eventFields interface{}
	if len(fields) > 1 {
		eventFields = fields
	} else if len(fields) == 1 {
		eventFields = fields[0]
	} else if len(fields) == 0 {
		eventFields = nil
	}

	eventId := atomic.AddUint64(&logger.lastEventId, 1)
	logEvent := event.NewEvent(eventId, level, message, eventFields)
	logger.mutex.RLock()
	for _, eventHandlerSpec := range logger.eventHandlerMap {
		if true { //TODO make dropping configurable per-handler
			select {
			case eventHandlerSpec.eventChannel <- logEvent:
				atomic.StoreUint64(&eventHandlerSpec.lastSentEventId, logEvent.Id)
			default:
				fmt.Fprintf(os.Stderr, "Unable to send event to handler. Buffer full. handler=%s\n", eventHandlerSpec.name)
				//TODO generate an event for this, but put in a time-last-dropped so we don't send the message to the handler which is dropping
				// basically if we are dropping, and we last dropped < X seconds ago, don't generate another "event dropped" message
			}
		} else {
			eventHandlerSpec.eventChannel <- logEvent
			atomic.StoreUint64(&eventHandlerSpec.lastSentEventId, logEvent.Id)
		}
	}
	logger.mutex.RUnlock()

	return eventId
}

// Emergency generates an event at the emergency level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Emergency(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Emergency, message, fields...)
}

// Alert generates an event at the alert level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Alert(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Alert, message, fields...)
}

// Critical generates an event at the critical level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Critical(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Critical, message, fields...)
}

// Error generates an event at the error level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Error(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Error, message, fields...)
}

// Warning generates an event at the warning level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Warning(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Warning, message, fields...)
}

// Notice generates an event at the notice level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Notice(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Notice, message, fields...)
}

// Info generates an event at the info level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Info(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Info, message, fields...)
}

// Debug generates an event at the debug level.
// It returns an event Id that can be used with Sync().
func (logger *Logger) Debug(message string, fields ...interface{}) uint64 {
	return logger.Event(event.Debug, message, fields...)
}

// Fatal generates an event at the critical level, and then exits the program with status 1
func (logger *Logger) Fatal(message string, fields ...interface{}) {
	logger.Critical(message, fields...)
	logger.Stop()
	exit(1)
}

// Sync blocks until the given event Id has been flushed out to all destinations.
func (logger *Logger) Sync(eventId uint64) {
	logger.mutex.RLock()
	for _, eventHandlerSpec := range logger.eventHandlerMap {
		if atomic.LoadUint64(&eventHandlerSpec.lastSentEventId) < eventId {
			// lastSentEventId wasn't incremented, meaning it was dropped. no point waiting for it
			continue
		}

		// wait for the lastProcessedEventId to become >= eventId
		eventHandlerSpec.lastProcessedEventIdCond.L.Lock()
		for eventHandlerSpec.lastProcessedEventId < eventId {
			eventHandlerSpec.lastProcessedEventIdCond.Wait()
		}
		eventHandlerSpec.lastProcessedEventIdCond.L.Unlock()
	}
	logger.mutex.RUnlock()
}
