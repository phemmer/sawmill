package sawmill

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"github.com/phemmer/sawmill/handler/syslog"
	"github.com/phemmer/sawmill/handler/writer"
)

type Fields map[string]interface{}

type Handler interface {
	Event(event *event.Event) error
}

type eventHandlerSpec struct {
	name               string
	levelMin, levelMax event.Level
	eventChannel       chan *event.Event
	finishChannel      chan bool

	lastSentEventId          uint64
	lastProcessedEventId     uint64
	lastProcessedEventIdCond *sync.Cond
}

type Logger struct {
	eventHandlerMap map[string]*eventHandlerSpec
	mutex           sync.RWMutex
	waitgroup       sync.WaitGroup
	lastEventId     uint64
}

func NewLogger() *Logger {
	return &Logger{
		eventHandlerMap: make(map[string]*eventHandlerSpec),
	}
}

func (logger *Logger) AddHandler(name string, handler Handler, levelMin event.Level, levelMax event.Level) {
	//TODO check name collision
	spec := &eventHandlerSpec{
		name:                     name,
		levelMin:                 levelMin,
		levelMax:                 levelMax,
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

func (logger *Logger) InitStdStreams() {
	var stdoutFormat, stderrFormat string
	if writer.IsTerminal(os.Stdout) {
		stdoutFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stdoutFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}
	if writer.IsTerminal(os.Stderr) {
		stderrFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stderrFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}

	stdoutHandler, _ := writer.NewEventWriter(os.Stdout, stdoutFormat) // eat the error. the only possible issue is if the template has format errors, and we're using the default, which is hard-coded
	logger.AddHandler("stdout", stdoutHandler, Debug, Notice)
	stderrHandler, _ := writer.NewEventWriter(os.Stderr, stderrFormat)
	logger.AddHandler("stderr", stderrHandler, Warning, Emergency)
}
func (logger *Logger) InitStdSyslog() error {
	syslogHandler, err := syslog.NewSyslogWriter("", "", 0, "")
	if err != nil {
		return err
	}
	logger.AddHandler("syslog", syslogHandler, Debug, Emergency)

	return nil
}

func (logger *Logger) Event(level event.Level, message string, fields interface{}) uint64 {
	eventId := atomic.AddUint64(&logger.lastEventId, 1)
	logEvent := event.NewEvent(eventId, level, message, fields)

	logger.mutex.RLock()
	for _, eventHandlerSpec := range logger.eventHandlerMap {
		if level > eventHandlerSpec.levelMin || level < eventHandlerSpec.levelMax { // levels are based off syslog levels, so the highest level (emergency) is `0`, and the min (debug) is `7`. This means our comparisons look weird
			continue
		}
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

func (logger *Logger) Emergency(message string, fields interface{}) uint64 {
	return logger.Event(event.Emergency, message, fields)
}

func (logger *Logger) Alert(message string, fields interface{}) uint64 {
	return logger.Event(event.Alert, message, fields)
}

func (logger *Logger) Critical(message string, fields interface{}) uint64 {
	return logger.Event(event.Critical, message, fields)
}

func (logger *Logger) Error(message string, fields interface{}) uint64 {
	return logger.Event(event.Error, message, fields)
}

func (logger *Logger) Warning(message string, fields interface{}) uint64 {
	return logger.Event(event.Warning, message, fields)
}

func (logger *Logger) Notice(message string, fields interface{}) uint64 {
	return logger.Event(event.Notice, message, fields)
}

func (logger *Logger) Info(message string, fields interface{}) uint64 {
	return logger.Event(event.Info, message, fields)
}

func (logger *Logger) Debug(message string, fields interface{}) uint64 {
	return logger.Event(event.Debug, message, fields)
}

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
