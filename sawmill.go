package sawmill

import (
	"os"

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

var logger *Logger

func DefaultLogger() *Logger {
	if logger == nil {
		logger = NewLogger()
		logger.InitStdStreams()
	}

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
	Critical(message, fields)
	Stop()
	os.Exit(1)
}

func Sync(eventId uint64) {
	DefaultLogger().Sync(eventId)
}

func Stop() {
	DefaultLogger().Stop()
}
