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

func (writer *standardStreamsWriter) Event(logEvent *event.Event) error {
	if logEvent.Level <= event.Warning { //TODO fix the ordering of the levels to be what is expected
		return writer.stderrWriter.Event(logEvent)
	}
	return writer.stdoutWriter.Event(logEvent)
}
