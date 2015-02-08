package sawmill

import (
  "fmt"
  "os"
  "syscall"
  "testing"
  "time"

  "github.com/phemmer/sawmill/event"
  "github.com/stretchr/testify/assert"
)



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
