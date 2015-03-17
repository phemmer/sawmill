package sawmill

import (
	"bufio"
	"io"
	"runtime"

	"github.com/phemmer/sawmill/event"
)

// NewWriter returns an io.Writer compatable object that can be used for traditional writing into sawmill.
//
// The main use case for this is to redirect the stdlib log package into sawmill. For example:
//  log.SetOutput(logger.NewWriter(sawmill.InfoLevel))
//  log.SetFlags(0) // sawmill does its own formatting
func (logger *Logger) NewWriter(level event.Level) io.WriteCloser {
	if level < 0 {
		level = event.Info
	}

	pipeReader, pipeWriter := io.Pipe()

	//TODO(.) I don't like goroutines, and there's no technical reason why this requires one.
	// The current implementation is just simpler.
	go logger.writerScanner(pipeReader, level)
	runtime.SetFinalizer(pipeWriter, writerFinalizer)

	return pipeWriter
}

func writerFinalizer(writer io.Closer) {
	writer.Close()
}

func (logger *Logger) writerScanner(reader io.ReadCloser, level event.Level) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logger.Event(level, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		logger.Event(event.Error, "Error while reading from input writer", Fields{"error": err})
	}
	reader.Close()
}
