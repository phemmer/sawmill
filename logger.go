package sawmill

import (
	"time"
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"github.com/phemmer/sawmill/hook"
	"github.com/phemmer/sawmill/hook/syslog"
	"os"
	"reflect"
	"fmt"
	"sync"
)

type Fields map[string]interface{}

type hookTableEntry struct {
	name string
	levelMin, levelMax event.Level
	eventChannel chan *event.Event
	finishChannel chan bool
}
type Logger struct {
	hookTable map[string]*hookTableEntry
	waitgroup sync.WaitGroup
}

func NewLogger() (*Logger) {
	return &Logger{
		hookTable: make(map[string]*hookTableEntry),
	}
}

func (logger *Logger) AddHook(name string, hook hook.Hook, levelMin event.Level, levelMax event.Level) {
	//TODO lock
	//TODO check name collision
	hookTableEntry := &hookTableEntry{
		name: name,
		levelMin: levelMin,
		levelMax: levelMax,
		eventChannel: make(chan *event.Event, 100),
		finishChannel: make(chan bool, 1),
	}

	logger.waitgroup.Add(1)
	go func(eventChannel chan *event.Event, callback func(*event.Event) error, waitgroup *sync.WaitGroup, finishChannel chan bool) {
		defer waitgroup.Done()
		for logEvent := range eventChannel {
			if logEvent == nil {
				break
			}
			callback(logEvent) //TODO error handler
		}
		finishChannel <- true
	}(hookTableEntry.eventChannel, hook.Event, &logger.waitgroup, hookTableEntry.finishChannel)

	logger.hookTable[name] = hookTableEntry
}
func (logger *Logger) RemoveHook(name string, wait bool) {
	hookTableEntry := logger.hookTable[name]
	if hookTableEntry == nil {
		// doesn't exist
		return
	}
	delete(logger.hookTable, name)
	hookTableEntry.eventChannel <- nil
	if !wait {
		return
	}
	<-hookTableEntry.finishChannel
}
func (logger *Logger) Stop() {
	for hookName, _ := range logger.hookTable {
		logger.RemoveHook(hookName, false)
	}
	logger.waitgroup.Wait() //TODO timeout?
}

func (logger *Logger) InitStdStreams() {
	var stdoutFormat, stderrFormat string
	if hook.IsTerminal(os.Stdout) {
		stdoutFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stdoutFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}
	if hook.IsTerminal(os.Stderr) {
		stderrFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stderrFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}

	stdoutHook, _ := hook.NewHookIOWriter(os.Stdout, stdoutFormat) // eat the error. the only possible issue is if the template has format errors, and we're using the default, which is hard-coded
	logger.AddHook("stdout", stdoutHook, Debug, Notice)
	stderrHook, _ := hook.NewHookIOWriter(os.Stdout, stderrFormat)
	logger.AddHook("stderr", stderrHook, Warning, Emergency)
}
func (logger *Logger) InitStdSyslog() (error) {
	syslogHook, err := syslog.New("", "", 0, "")
	if err != nil {
		return err
	}
	logger.AddHook("syslog", syslogHook, Debug, Emergency)

	return nil
}

func (logger *Logger) Event(level event.Level, message string, fields interface{}) {
	fieldsCopy := deStruct(fields)
	logEvent := &event.Event{
		Time: time.Now(),
		Level: level,
		Message: message,
		Fields: fieldsCopy,
	}
	//TODO lock table, copy it, release lock, iterate over copy
	for _, hookTableEntry := range logger.hookTable {
		if level > hookTableEntry.levelMin || level < hookTableEntry.levelMax { // levels are based off syslog levels, so the highest level (emergency) is `0`, and the min (debug) is `7`. This means our comparisons look weird
			continue
		}
		select {
		case hookTableEntry.eventChannel <- logEvent:
		default:
			fmt.Fprintf(os.Stderr, "Unable to send event to hook. Buffer full. hook=%s\n", hookTableEntry.name)
			//TODO generate an event for this, but put in a time-last-dropped so we don't send the message to the hook which is dropping
			// basically if we are dropping, and we last dropped < X seconds ago, don't generate another "event dropped" message
		}
	}
}



func deStruct(obj interface{}) (interface{}) {
	value := reflect.ValueOf(obj)
	for value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() == reflect.Struct {
		result := make(map[string]interface{})
		structType := reflect.TypeOf(value.Interface())
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if ! field.CanInterface() { // skip if it's unexported
				continue
			}
			k := structType.Field(i).Name
			result[k] = deStruct(field.Interface())
		}
		return result
	} else if value.Kind() == reflect.Map {
		result := make(map[interface{}]interface{})
		for _, kValue := range value.MapKeys() {
			vValue := value.MapIndex(kValue)
			k := kValue.Interface()
			result[deStruct(k)] = deStruct(vValue.Interface())
		}
		return result
	} else if value.Kind() == reflect.Array || value.Kind() == reflect.Slice {
		var result []interface{}
		for v := range value.Interface().([]interface{}) {
			result = append(result, deStruct(v))
		}
		return result
	}
	// scalar
	return value.Interface()
}
