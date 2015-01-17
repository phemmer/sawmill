package sawmill

import (
	"time"
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/hook"
	"github.com/phemmer/sawmill/formatter"
	"github.com/mitchellh/copystructure"
	"os"
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

type Fields map[string]interface{}

////////////////////////////////////////

type hookTableEntry struct {
	name string
	levelMin, levelMax event.Level
	hook hook.Hook
}
type Logger struct {
	hookTable []*hookTableEntry
}

var logger *Logger

func NewLogger() (*Logger) {
	return &Logger{}
}

func (logger *Logger) AddHook(name string, hook hook.Hook, levelMin event.Level, levelMax event.Level) {
	//TODO lock
	logger.hookTable = append(logger.hookTable, &hookTableEntry{
		name: name,
		hook: hook,
		levelMin: levelMin,
		levelMax: levelMax,
	})
}

func (logger *Logger) InitStdStreams() {
	logger.AddHook("stdout", hook.NewHookIOWriter(os.Stdout, formatter.NewTextFormatter()), Debug, Notice)
	logger.AddHook("stderr", hook.NewHookIOWriter(os.Stderr, formatter.NewTextFormatter()), Warning, Emergency)
}

func Event(level event.Level, message string, fields interface{}) {
	if logger == nil {
		logger = NewLogger()
		logger.InitStdStreams()
	}
	logger.Event(level, message, fields)
}
func (logger *Logger) Event(level event.Level, message string, fields interface{}) {
	fieldsCopy, _ := copystructure.Copy(fields)
	logEvent := &event.Event{
		Timestamp: time.Now(),
		Level: level,
		Message: message,
		Fields: fieldsCopy,
	}
	//TODO lock table, copy it, release lock, iterate over copy
	for _, hookTableEntry := range logger.hookTable {
		if level > hookTableEntry.levelMin || level < hookTableEntry.levelMax { // levels are based off syslog levels, so the highest level (emergency) is `0`, and the min (debug) is `7`. This means our comparisons look weird
			continue
		}
		hookTableEntry.hook.Event(logEvent)
	}
	//fmt.Printf("level=%d message=%s fields=%s:%#v\n", level, message, reflect.TypeOf(fields), fields)
}
