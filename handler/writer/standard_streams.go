package writer

import (
	"github.com/phemmer/sawmill/event"
	"os"
)

type standardStreamsHandler struct {
	stdoutWriter *WriterHandler
	stderrWriter *WriterHandler
}

// NewStandardStreamsHandler is a convenience function for constructing a new handler which sends to STDOUT/STDERR.
// If the output is sent to a TTY, the format is event.ConsoleColorFormat. Otherwise it is event.ConsoleNocolorFormat. The only difference between the two are the use of color escape codes.
func NewStandardStreamsHandler() *standardStreamsHandler {
	var stdoutFormat, stderrFormat string
	if IsTerminal(os.Stdout) {
		stdoutFormat = event.ConsoleColorFormat
	} else {
		stdoutFormat = event.ConsoleNocolorFormat
	}
	if IsTerminal(os.Stderr) {
		stderrFormat = event.ConsoleColorFormat
	} else {
		stderrFormat = event.ConsoleNocolorFormat
	}

	handler := &standardStreamsHandler{}

	// Discard the errors in the following.
	// The only possible issue is if the template has format errors, and we're using the default, which is hard-coded.
	handler.stdoutWriter, _ = New(os.Stdout, stdoutFormat)
	handler.stderrWriter, _ = New(os.Stderr, stderrFormat)

	return handler
}

// Event accepts an event and sends it to the appropriate output stream based on the event's level.
// If the level is warning or higher, it is sent to STDERR. Otherwise it is sent to STDOUT.
func (handler *standardStreamsHandler) Event(logEvent *event.Event) error {
	if logEvent.Level >= event.Warning {
		return handler.stderrWriter.Event(logEvent)
	}
	return handler.stdoutWriter.Event(logEvent)
}
