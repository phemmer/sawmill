/*
Sawmill is an asynchronous, structured, log event handler.

Asynchronous: Sawmill does not block execution waiting for the log message to be delivered to the destination (e.g. STDOUT).
Because of this asynchronous processing, it is critical that you add a `defer sawmill.Stop()` at the top of your `main()`. This will ensure that when the program exits, it waits for any pending log events to flush out to their destination.

Structured: Sawmill places a heavy emphasis on events with ancillary data.
A log event (e.g. `sawmill.Error()`) should have a simple string that is an event description, such as "Image processing failed", and then a map or struct included with details on the event.

----

The base package provides a default logger that will send events to STDOUT or STDERR as appropriate. This default logger is shared by all consumers of the package.

*/
package sawmill

import (
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/handler/filter"
)

// these are copied here for convenience
const (
	DebugLevel     = event.Debug
	InfoLevel      = event.Info
	NoticeLevel    = event.Notice
	WarningLevel   = event.Warning
	ErrorLevel     = event.Error
	CriticalLevel  = event.Critical
	AlertLevel     = event.Alert
	EmergencyLevel = event.Emergency
)

var defaultLoggerValue atomic.Value
var defaultLoggerMutex sync.Mutex

// DefaultLogger returns a common *Logger object that is shared among all consumers of the package. It is used implicitly by all the package level helper function (Event, Emergency, etc)
func DefaultLogger() *Logger {
	// The *Logger object is not created or intialized until after the first call to this function. This is because each Logger starts a goroutine, and we don't want to start a goroutine simply because the package was imported.
	var logger *Logger

	loggerValue := defaultLoggerValue.Load()
	if loggerValue == nil {
		defaultLoggerMutex.Lock()
		loggerValue = defaultLoggerValue.Load()
		if loggerValue == nil {
			logger = NewLogger()
			logger.InitStdStreams()
			defaultLoggerValue.Store(logger)
		}
		defaultLoggerMutex.Unlock()
		loggerValue = defaultLoggerValue.Load()
	}

	logger = loggerValue.(*Logger)

	return logger
}

// SetStackMinLevel sets the minimum level at which to include a stack trace
// in events.
func SetStackMinLevel(level event.Level) {
	DefaultLogger().SetStackMinLevel(level)
}

// GetStackMinLevel gets the minimum level at which to include a stack trace
// in events.
func GetStackMinLevel() event.Level {
	return DefaultLogger().GetStackMinLevel()
}

// AddHandler registers a new destination handler with the logger.
//
// The name parameter is a unique identifier so that the handler can be targeted with RemoveHandler().
//
// If a handler with the same name already exists, it will be replaced by the new one.
// During replacement, the function will block waiting for any pending events to be flushed to the old handler.
func AddHandler(name string, handler Handler) {
	DefaultLogger().AddHandler(name, handler)
}

// RemoveHandler removes the named handler from the logger, preventing any further events from being sent to it.
// The wait parameter will result in the function blocking until all events queued for the handler have finished processing.
func RemoveHandler(name string, wait bool) {
	DefaultLogger().RemoveHandler(name, wait)
}

// GetHandler retrieves the handler with the given name.
// Returns nil if no such handler exists.
func GetHandler(name string) Handler {
	return DefaultLogger().GetHandler(name)
}

// FilterHandler is a convience wrapper for filter.New().
//
// Example usage:
//  stdStreamsHandler := sawmill.GetHandler("stdStreams")
//  stdStreamsHandler = sawmill.FilterHandler(stdStreamsHandler).LevelMin(sawmill.ErrorLevel)
//  sawmill.AddHandler("stdStreams", stdStreamsHandler)
func FilterHandler(handler Handler, filterFuncs ...filter.FilterFunc) *filter.FilterHandler {
	return DefaultLogger().FilterHandler(handler, filterFuncs...)
}

// InitStdStreams is a convience function to register a STDOUT/STDERR handler with the logger.
//
// The is automatically invoked on the default package level logger, and should not normally be called.
func InitStdStreams() {
	DefaultLogger().InitStdStreams()
}

// InitStdSyslog is a convenience function to register a syslog handler with the logger.
//
// The handler is added with the name 'syslog'
func InitStdSyslog() error {
	return DefaultLogger().InitStdSyslog()
}

// Event queues a message at the given level.
// Additional fields may be provided, which will be recursively copied at the time of the function call, and provided to the destination output handler.
// It returns an event Id that can be used with Sync().
func Event(level event.Level, message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(level, message, fields...)
}

// SendEvent queues the given event.
// The event's `Id` field will be updated with a value that can be used by
// Sync(). This value is also provided as the return value for convenience.
func SendEvent(logEvent *event.Event) uint64 {
	return DefaultLogger().SendEvent(logEvent)
}

// Emergency generates an event at the emergency level.
// It returns an event Id that can be used with Sync().
func Emergency(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Emergency, message, fields...)
}

// Alert generates an event at the alert level.
// It returns an event Id that can be used with Sync().
func Alert(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Alert, message, fields...)
}

// Critical generates an event at the critical level.
// It returns an event Id that can be used with Sync().
func Critical(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Critical, message, fields...)
}

// Error generates an event at the error level.
// It returns an event Id that can be used with Sync().
func Error(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Error, message, fields...)
}

// Warning generates an event at the warning level.
// It returns an event Id that can be used with Sync().
func Warning(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Warning, message, fields...)
}

// Notice generates an event at the notice level.
// It returns an event Id that can be used with Sync().
func Notice(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Notice, message, fields...)
}

// Info generates an event at the info level.
// It returns an event Id that can be used with Sync().
func Info(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Info, message, fields...)
}

// Debug generates an event at the debug level.
// It returns an event Id that can be used with Sync().
func Debug(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Debug, message, fields...)
}

// Fatal generates an event at the critical level, and then exits the program with status 1
func Fatal(message string, fields ...interface{}) {
	Critical(message, fields...)
	Stop()
	os.Exit(1)
}

// Sync blocks until the given event Id has been flushed out to all destinations.
func Sync(eventId uint64) {
	DefaultLogger().Sync(eventId)
}

// SetSync controls synchronous event mode. When set to true, a function call
// to generate an event does not return until the event has been processed.
func SetSync(enabled bool) {
	DefaultLogger().SetSync(enabled)
}

// GetSync indicates whether syncronous mode is enabled.
func GetSync() bool {
	return DefaultLogger().GetSync()
}

// Stop removes all destinations on the logger, and waits for any pending events to flush to their destinations.
func Stop() {
	DefaultLogger().checkPanic(recover())
	DefaultLogger().Stop()
}

// CheckPanic is used to check for panics and log them when encountered.
// The function must be executed via defer.
// CheckPanic will not halt the panic. After logging, the panic will be passed
// through.
func CheckPanic() {
	DefaultLogger().checkPanic(recover())
}

// NewWriter returns an io.WriteCloser compatable object that can be used for traditional writing into sawmill.
//
// The main use case for this is to redirect the stdlib log package into sawmill. For example:
//  log.SetOutput(sawmill.NewWriter(sawmill.InfoLevel))
//  log.SetFlags(0) // sawmill does its own formatting
func NewWriter(level event.Level) io.WriteCloser {
	return DefaultLogger().NewWriter(level)
}
