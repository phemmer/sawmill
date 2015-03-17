/*
Sawmill is an asynchronous, structured, log event handler.

Asynchronous: Sawmill does not block execution waiting for the log message to be delivered to the destination (e.g. STDOUT).
Because of this asynchronous processing, it is critical that you add a `defer sawmill.Stop()` at the top of your `main()`. This will ensure that when the program exits, it waits for any pending log events to flush out to their destination.

And 'structured' means that sawmill places a heavy emphasis on events with ancillary data.
A log event (e.g. `sawmill.Error()`) should have a simple string that is an event description, such as "Image processing failed", and then a map or struct included with details on the event.

*/
package sawmill

import (
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/phemmer/sawmill/event"
)

// these are copied here for convenience
const (
	EmergencyLevel = event.Emergency
	AlertLevel     = event.Alert
	CriticalLevel  = event.Critical
	ErrorLevel     = event.Error
	WarningLevel   = event.Warning
	NoticeLevel    = event.Notice
	InfoLevel      = event.Info
	DebugLevel     = event.Debug
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

func Event(level event.Level, message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(level, message, fields...)
}

func Emergency(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Emergency, message, fields...)
}

func Alert(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Alert, message, fields...)
}

func Critical(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Critical, message, fields...)
}

func Error(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Error, message, fields...)
}

func Warning(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Warning, message, fields...)
}

func Notice(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Notice, message, fields...)
}

func Info(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Info, message, fields...)
}

func Debug(message string, fields ...interface{}) uint64 {
	return DefaultLogger().Event(event.Debug, message, fields...)
}

func Fatal(message string, fields ...interface{}) {
	Critical(message, fields...)
	Stop()
	os.Exit(1)
}

func Sync(eventId uint64) {
	DefaultLogger().Sync(eventId)
}

func Stop() {
	DefaultLogger().Stop()
}

func NewWriter(level event.Level) io.WriteCloser {
	return DefaultLogger().NewWriter(level)
}
