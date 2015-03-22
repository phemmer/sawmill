package writer

import (
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"os"
)

type standardStreamsWriter struct {
	stdoutWriter *EventWriter
	stderrWriter *EventWriter
}

// NewStandardStreamsWriter is a convenience function for constructing a new writer which sends to STDOUT/STDERR.
// If the output is sent to a TTY, the format is formatter.CONSOLE_COLOR_FORMAT. Otherwise it is formatter.CONSOLE_NOCOLOR_FORMAT. The only difference between the two are the use of color escape codes.
func NewStandardStreamsWriter() *standardStreamsWriter {
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

	writer := &standardStreamsWriter{}

	// Discard the errors in the following.
	// The only possible issue is if the template has format errors, and we're using the default, which is hard-coded.
	writer.stdoutWriter, _ = NewEventWriter(os.Stdout, stdoutFormat)
	writer.stderrWriter, _ = NewEventWriter(os.Stderr, stderrFormat)

	return writer
}

// Event accepts an event and sends it to the appropriate output stream based on the event's level.
// If the level is warning or higher, it is sent to STDERR. Otherwise it is sent to STDOUT.
func (writer *standardStreamsWriter) Event(logEvent *event.Event) error {
	if logEvent.Level >= event.Warning {
		return writer.stderrWriter.Event(logEvent)
	}
	return writer.stdoutWriter.Event(logEvent)
}
