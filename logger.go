package sawmill

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/handler/filter"
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
	handler       Handler
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
	stackMinLevel   int32 // we store this as int32 instead of event.Level so that we can use atomic
	mutex           sync.RWMutex
	waitgroup       sync.WaitGroup
	lastEventId     uint64
	syncEnabled     uint32
}

// NewLogger constructs a Logger.
// The new Logger will not have any registered handlers.
//
// By default events will not include a stack trace. If any destination
// handler makes use of a stack trace, call SetStackMinLevel on the logger.
func NewLogger() *Logger {
	return &Logger{
		eventHandlerMap: make(map[string]*eventHandlerSpec),
		stackMinLevel:   int32(event.Emergency) + 1,
	}
}

// SetStackMinLevel sets the minimum level at which to include a stack trace
// in events.
func (logger *Logger) SetStackMinLevel(level event.Level) {
	atomic.StoreInt32(&logger.stackMinLevel, int32(level))
}

// GetStackMinLevel gets the minimum level at which to include a stack trace
// in events.
func (logger *Logger) GetStackMinLevel() event.Level {
	return event.Level(atomic.LoadInt32(&logger.stackMinLevel))
}

// AddHandler registers a new destination handler with the logger.
//
// The name parameter is a unique identifier so that the handler can be targeted with RemoveHandler().
//
// If a handler with the same name already exists, it will be replaced by the new one.
// During replacement, the function will block waiting for any pending events to be flushed to the old handler.
func (logger *Logger) AddHandler(name string, handler Handler) {
	spec := &eventHandlerSpec{
		name:                     name,
		handler:                  handler,
		eventChannel:             make(chan *event.Event, 100),
		finishChannel:            make(chan bool, 1),
		lastProcessedEventIdCond: sync.NewCond(&sync.Mutex{}),
	}

	logger.waitgroup.Add(1)
	go handlerDriver(spec, handler, &logger.waitgroup)

	logger.mutex.Lock()
	oldSpec := logger.eventHandlerMap[name]
	logger.eventHandlerMap[name] = spec
	logger.mutex.Unlock()

	//TODO we need a way to leave the handler in the map while letting it drain.
	// With the current code, we remove the handler and wait. But if someone
	// calls Sync(), it won't know about this handler any more.
	//
	// We could add the handler into the map under an alternate name, and remove
	// it once drained.
	// Or we could have 2 maps. Add another one for "pending removal" handlers.
	if oldSpec != nil {
		oldSpec.eventChannel <- nil
		<-oldSpec.finishChannel
	}
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

// GetHandler retrieves the handler with the given name.
// Returns nil if no such handler exists.
func (logger *Logger) GetHandler(name string) Handler {
	logger.mutex.RLock()
	handlerSpec := logger.eventHandlerMap[name]
	logger.mutex.RUnlock()

	if handlerSpec == nil {
		return nil
	}
	return handlerSpec.handler
}

// FilterHandler is a convience wrapper for filter.New().
//
// Example usage:
//  stdStreamsHandler := logger.GetHandler("stdStreams")
//  stdStreamsHandler = logger.FilterHandler(stdStreamsHandler).LevelMin(sawmill.ErrorLevel)
//  logger.AddHandler("stdStreams", stdStreamsHandler)
func (logger *Logger) FilterHandler(handler Handler, filterFuncs ...filter.FilterFunc) *filter.FilterHandler {
	return filter.New(handler, filterFuncs...)
}

// Stop removes all destination handlers on the logger, and waits for any pending events to flush out.
func (logger *Logger) Stop() {
	logger.checkPanic(recover())

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

// CheckPanic is used to check for panics and log them when encountered.
// The function must be executed via defer.
// CheckPanic will not halt the panic. After logging, the panic will be passed
// through.
func (logger *Logger) CheckPanic() {
	// recover only works when called in the function that was deferred, not
	// recurisvely. But the code is shared by several other functions.
	logger.checkPanic(recover())
}
func (logger *Logger) checkPanic(err interface{}) {
	if err == nil {
		return
	}
	logger.Sync(logger.Critical("panic", Fields{"error": err}))
	panic(err)
}

// InitStdStreams is a convience function to register a STDOUT/STDERR handler with the logger.
//
// The handler is added with the name 'stdStreams'
func (logger *Logger) InitStdStreams() {
	logger.AddHandler("stdStreams", writer.NewStandardStreamsHandler())
}

// InitStdSyslog is a convenience function to register a syslog handler with the logger.
//
// The handler is added with the name 'syslog'
func (logger *Logger) InitStdSyslog() error {
	syslogHandler, err := syslog.New("", "", 0, "")
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

	getStack := int32(level) >= atomic.LoadInt32(&logger.stackMinLevel)
	//TODO do we want to just remove the id param from event.New()?
	logEvent := event.New(0, level, message, eventFields, getStack)

	return logger.SendEvent(logEvent)
}

// SendEvent queues the given event.
// The event's `Id` field will be updated with a value that can be used by
// Sync(). This value is also provided as the return value for convenience.
func (logger *Logger) SendEvent(logEvent *event.Event) uint64 {
	logEvent.Id = atomic.AddUint64(&logger.lastEventId, 1)

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

	if logger.GetSync() {
		logger.Sync(logEvent.Id)
	}

	return logEvent.Id
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

// SetSync controls synchronous event mode. When set to true, a function call
// to generate an event does not return until the event has been processed.
func (logger *Logger) SetSync(enabled bool) {
	if enabled {
		atomic.StoreUint32(&logger.syncEnabled, 1)
	} else {
		atomic.StoreUint32(&logger.syncEnabled, 0)
	}
}

// GetSync indicates whether syncronous mode is enabled.
func (logger *Logger) GetSync() bool {
	return atomic.LoadUint32(&logger.syncEnabled) == 1
}
