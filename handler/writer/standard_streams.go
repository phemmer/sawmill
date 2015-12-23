package writer

import (
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"os"
)

type StandardStreamsHandler struct {
	stdoutWriter *WriterHandler
	stderrWriter *WriterHandler
}

// NewStandardStreamsHandler is a convenience function for constructing a new handler which sends to STDOUT/STDERR.
// If the output is sent to a TTY, the format is formatter.CONSOLE_COLOR_FORMAT. Otherwise it is formatter.CONSOLE_NOCOLOR_FORMAT. The only difference between the two are the use of color escape codes.
func NewStandardStreamsHandler() *StandardStreamsHandler {
	var stdoutFormat, stderrFormat string
	if IsTerminal(os.Stdout) {
		stdoutFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stdoutFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}
	if IsTerminal(os.Stderr) {
		stderrFormat = formatter.CONSOLE_COLOR_FORMAT
	} else {
		stderrFormat = formatter.CONSOLE_NOCOLOR_FORMAT
	}

	handler := &StandardStreamsHandler{}

	// Discard the errors in the following.
	// The only possible issue is if the template has format errors, and we're using the default, which is hard-coded.
	handler.stdoutWriter, _ = New(os.Stdout, stdoutFormat)
	handler.stderrWriter, _ = New(os.Stderr, stderrFormat)

	return handler
}

// Event accepts an event and sends it to the appropriate output stream based on the event's level.
// If the level is warning or higher, it is sent to STDERR. Otherwise it is sent to STDOUT.
func (handler *StandardStreamsHandler) Event(logEvent *event.Event) error {
	if logEvent.Level >= event.Warning {
		return handler.stderrWriter.Event(logEvent)
	}
	return handler.stdoutWriter.Event(logEvent)
}
