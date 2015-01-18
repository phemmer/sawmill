package sawmill

import (
	"github.com/phemmer/sawmill/event"
)

// these are copied here for convenience
const (
	Emergency, Emerg = event.Emergency, event.Emerg
	Alert, Alrt = event.Alert, event.Alrt
	Critical, Crit = event.Critical, event.Crit
	Error, Err = event.Error, event.Err
	Warning, Warn = event.Warning, event.Warn
	Notice = event.Notice
	Info = event.Info
	Debug, Dbg = event.Debug, event.Debug
)

var logger *Logger
func NewLogger() (*Logger) {
	return &Logger{}
}
func Event(level event.Level, message string, fields interface{}) {
	if logger == nil {
		logger = NewLogger()
		logger.InitStdStreams()
	}
	logger.Event(level, message, fields)
}
