package sawmill

import (
	"time"
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"github.com/phemmer/sawmill/handler"
	"github.com/phemmer/sawmill/handler/syslog"
	"os"
	"reflect"
	"fmt"
	"sync"
)

type Fields map[string]interface{}

type eventHandlerSpec struct {
	name string
	levelMin, levelMax event.Level
	eventChannel chan *event.Event
	finishChannel chan bool
}
type Logger struct {
	eventHandlerMap map[string]*eventHandlerSpec
	waitgroup sync.WaitGroup
}

func NewLogger() (*Logger) {
	return &Logger{
		eventHandlerMap: make(map[string]*eventHandlerSpec),
	}
}

func (logger *Logger) AddHandler(name string, eventHandler handler.Handler, levelMin event.Level, levelMax event.Level) {
	//TODO lock
	//TODO check name collision
	eventHandlerSpec := &eventHandlerSpec{
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
	}(eventHandlerSpec.eventChannel, eventHandler.Event, &logger.waitgroup, eventHandlerSpec.finishChannel)

	logger.eventHandlerMap[name] = eventHandlerSpec
}
func (logger *Logger) RemoveHandler(name string, wait bool) {
	eventHandlerSpec := logger.eventHandlerMap[name]
	if eventHandlerSpec == nil {
		// doesn't exist
		return
	}
	delete(logger.eventHandlerMap, name)
	eventHandlerSpec.eventChannel <- nil
	if !wait {
		return
	}
	<-eventHandlerSpec.finishChannel
}
func (logger *Logger) Stop() {
	for handlerName, _ := range logger.eventHandlerMap {
		logger.RemoveHandler(handlerName, false)
	}
	logger.waitgroup.Wait() //TODO timeout?
}

func (logger *Logger) InitStdStreams() {
	var stdoutFormat, stderrFormat string
	if handler.IsTerminal(os.Stdout) {
		stdoutFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stdoutFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}
	if handler.IsTerminal(os.Stderr) {
		stderrFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stderrFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}

	stdoutHandler, _ := handler.NewEventIOWriter(os.Stdout, stdoutFormat) // eat the error. the only possible issue is if the template has format errors, and we're using the default, which is hard-coded
	logger.AddHandler("stdout", stdoutHandler, Debug, Notice)
	stderrHandler, _ := handler.NewEventIOWriter(os.Stdout, stderrFormat)
	logger.AddHandler("stderr", stderrHandler, Warning, Emergency)
}
func (logger *Logger) InitStdSyslog() (error) {
	syslogHandler, err := syslog.New("", "", 0, "")
	if err != nil {
		return err
	}
	logger.AddHandler("syslog", syslogHandler, Debug, Emergency)

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
	for _, eventHandlerSpec := range logger.eventHandlerMap {
		if level > eventHandlerSpec.levelMin || level < eventHandlerSpec.levelMax { // levels are based off syslog levels, so the highest level (emergency) is `0`, and the min (debug) is `7`. This means our comparisons look weird
			continue
		}
		select {
		case eventHandlerSpec.eventChannel <- logEvent:
		default:
			fmt.Fprintf(os.Stderr, "Unable to send event to handler. Buffer full. handler=%s\n", eventHandlerSpec.name)
			//TODO generate an event for this, but put in a time-last-dropped so we don't send the message to the handler which is dropping
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
