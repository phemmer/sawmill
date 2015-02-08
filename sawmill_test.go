package sawmill

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/phemmer/sawmill/event"
	"github.com/stretchr/testify/assert"
)

type channelHandler struct {
	channel chan *event.Event
}

func NewChannelHandler() *channelHandler {
	return &channelHandler{
		channel: make(chan *event.Event),
	}
}
func (handler *channelHandler) Event(logEvent *event.Event) error {
	//fmt.Printf("Sending to channel: %#v\n", logEvent)
	handler.channel <- logEvent
	return nil
}
func (handler *channelHandler) Next(timeout time.Duration) *event.Event {
	var logEvent *event.Event
	if timeout == 0 {
		select {
		case logEvent = <-handler.channel:
		default:
		}
	} else {
		select {
		case logEvent = <-handler.channel:
		case <-time.After(time.Second * timeout):
		}
	}
	//fmt.Printf("Received from channel: %#v\n", logEvent)
	return logEvent
}

type captureHandler struct {
	events []*event.Event
	mutex  sync.Mutex
}

func (handler *captureHandler) Event(logEvent *event.Event) error {
	handler.mutex.Lock()
	handler.events = append(handler.events, logEvent)
	handler.mutex.Unlock()
	return nil
}
func (handler *captureHandler) Last() *event.Event {
	handler.mutex.Lock()
	logEvent := handler.events[len(handler.events)-1]
	handler.mutex.Unlock()
	return logEvent
}

func CaptureEvents() *captureHandler {
	logger := DefaultLogger()
	logger.RemoveHandler("stdout", false)
	logger.RemoveHandler("stderr", false)
	handler := &captureHandler{}
	logger.AddHandler("testcap", handler, DebugLevel, EmergencyLevel)
	return handler
}

func CaptureStream(file *os.File) (*os.File, func(), error) {
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	bkupFD, err := syscall.Dup(int(file.Fd()))
	if err != nil {
		pipeW.Close()
		pipeR.Close()
		return nil, nil, err
	}

	err = syscall.Dup2(int(pipeW.Fd()), int(file.Fd()))
	if err != nil {
		syscall.Close(bkupFD)
		pipeW.Close()
		pipeR.Close()
	}

	cleanFunc := func() {
		syscall.Dup2(bkupFD, int(file.Fd()))
		syscall.Close(bkupFD)
		pipeW.Close()
		pipeR.Close()
	}

	return pipeR, cleanFunc, nil
}

////////////////////////////////////////

func TestDefaultLogger(t *testing.T) {
	assert.NotEqual(t, DefaultLogger(), nil)
	assert.Equal(t, DefaultLogger(), DefaultLogger())
}

func TestEvent(t *testing.T) {
	testEventStream(t, NoticeLevel, os.Stdout, "stdout")
	testEventStream(t, WarningLevel, os.Stderr, "stderr")
}
func testEventStream(t *testing.T, level event.Level, stream *os.File, label string) {
	newStream, newStreamClose, err := CaptureStream(stream)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("Failed to capture %s. Cannot perform test. %s\n", label, err))
		return
	}
	defer newStreamClose()

	DefaultLogger().InitStdStreams() // re-open streams so that colors are off

	buf := make([]byte, 1024)

	message := fmt.Sprintf("TestEvent %s", label)

	Event(level, message, Fields{"stream": label})

	go func() { time.Sleep(time.Second); stream.Write([]byte{0}) }() // so that if it didn't go to the right stream, the Read() below won't block forever
	newStream.Read(buf)

	assert.Contains(t, string(buf), message)
	assert.Contains(t, string(buf), fmt.Sprintf("stream=%s", label))
}

func TestEmergency(t *testing.T) {
	testLevel(t, "Emergency", Emergency, os.Stderr)
}
func TestAlert(t *testing.T) {
	testLevel(t, "Alert", Alert, os.Stderr)
}
func TestCritical(t *testing.T) {
	testLevel(t, "Critical", Critical, os.Stderr)
}
func TestError(t *testing.T) {
	testLevel(t, "Error", Error, os.Stderr)
}
func TestWarning(t *testing.T) {
	testLevel(t, "Warning", Warning, os.Stderr)
}
func TestNotice(t *testing.T) {
	testLevel(t, "Notice", Notice, os.Stdout)
}
func TestInfo(t *testing.T) {
	testLevel(t, "Info", Info, os.Stdout)
}
func TestDebug(t *testing.T) {
	testLevel(t, "Debug", Debug, os.Stdout)
}
func testLevel(t *testing.T, levelString string, levelFunc func(string, ...interface{}) uint64, stream *os.File) {
	newStream, newStreamClose, err := CaptureStream(stream)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("Failed to capture stream for %s. Cannot perform test. %s\n", levelString, err))
		return
	}
	defer newStreamClose()

	DefaultLogger().InitStdStreams() // re-open streams so that colors are off

	buf := make([]byte, 1024)

	message := fmt.Sprintf("TestLevel %s", levelString)

	levelFunc(message, Fields{"level": levelString})

	go func() { time.Sleep(time.Second); stream.Write([]byte{0}) }() // so that if it didn't go to the right stream, the Read() below won't block forever
	newStream.Read(buf)

	assert.Contains(t, string(buf), message)
	assert.Contains(t, string(buf), fmt.Sprintf("level=%s", levelString))
}
