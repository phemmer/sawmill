package writer

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/phemmer/sawmill/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStandardStreamsHandler(t *testing.T) {
	_, pipeW, err := os.Pipe()
	require.NoError(t, err)
	defer func(f *os.File) { os.Stdout = f }(os.Stdout)
	os.Stdout = pipeW
	defer func(f *os.File) { os.Stderr = f }(os.Stderr)
	os.Stderr = pipeW

	h := NewStandardStreamsHandler()
	assert.Equal(t, h.stdoutWriter.Output, pipeW)
	//TODO find a way to check that the right template was used
	// The problem is that h.stderrWriter.Template is a parsed template, not the original string, so we can't compare.
	assert.Equal(t, h.stderrWriter.Output, pipeW)
	//TODO find a way to check that the right template was used
}

func TestStandardStreamsHandler_Event(t *testing.T) {
	wg := sync.WaitGroup{}

	pipeOutR, pipeOutW, err := os.Pipe()
	defer pipeOutW.Close()
	require.NoError(t, err)
	defer func(f *os.File) { os.Stdout = f }(os.Stdout)
	os.Stdout = pipeOutW
	outbuf := bytes.NewBuffer(nil)
	wg.Add(1)
	go func() { io.Copy(outbuf, pipeOutR); wg.Done() }()

	pipeErrR, pipeErrW, err := os.Pipe()
	defer pipeErrW.Close()
	require.NoError(t, err)
	defer func(f *os.File) { os.Stderr = f }(os.Stderr)
	os.Stderr = pipeErrW
	errbuf := bytes.NewBuffer(nil)
	wg.Add(1)
	go func() { io.Copy(errbuf, pipeErrR); wg.Done() }()

	h := NewStandardStreamsHandler()
	e := event.New(0, event.Info, "TestStandardStreamsHandler_Event info", nil, false)
	err = h.Event(e)
	assert.NoError(t, err)

	e = event.New(0, event.Error, "TestStandardStreamsHandler_Event error", nil, false)
	err = h.Event(e)
	assert.NoError(t, err)

	pipeOutW.Close()
	pipeErrW.Close()
	wg.Wait()

	assert.Contains(t, outbuf.String(), "TestStandardStreamsHandler_Event info")
	assert.Contains(t, errbuf.String(), "TestStandardStreamsHandler_Event error")
}
