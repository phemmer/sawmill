/*
The writer package is an event handler responsible for sending events to a generic IO writer.

Examples:

STDOUT/STDERR

 logger := sawmill.NewLogger()
 logger.AddHandler("stdStreams", writer.NewStandardStreamsHandler())

 # will go to STDERR
 logger.Warning("foo", sawmill.Fields{"bar": "baz"})

 # will go to STDOUT
 logger.Info("foo", sawmill.Fiels{"pop": "tart"})

 logger.Stop()

File appender

 logger := sawmill.NewLogger()
 h, err := writer.Append("/var/log/foo", 0600, "")
 if err != nil {
 	sawmill.Panic("error opening log file", sawmill.Fields{"error": err, "path": "/var/log/foo"})
 }
 logger.AddHandler("logfile", h)

 logger.Info("FOO!", sawmill.Fields{"bar": "baz"})
*/
package writer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"golang.org/x/crypto/ssh/terminal"
)

// IsTerminal returns whether the given stream (File) is attached to a TTY.
func IsTerminal(stream interface {
	Fd() uintptr
}) bool {
	return terminal.IsTerminal(int(stream.Fd()))
}

// WriterHandler is responsible for converting an event into text using a template, and then sending that text to an io.Writer.
type WriterHandler struct {
	Output   io.Writer
	Template *template.Template
}

// New constructs a new WriterHandler handler.
// templateString must be a template supported by the sawmill/event/formatter package.
// If the templateString is empty, the WriterHandler will use sawmill/event/formatter.SIMPLE_FORMAT.
func New(output io.Writer, templateString string) (*WriterHandler, error) {
	if templateString == "" {
		templateString = formatter.SIMPLE_FORMAT
	}
	formatterTemplate, err := template.New("").Parse(templateString)
	if err != nil {
		fmt.Printf("Error parsing template: %s", err) //TODO send message somewhere else?
		return nil, err
	}
	handler := &WriterHandler{
		Output:   output,
		Template: formatterTemplate,
	}
	return handler, nil
}

// Append constructs a new WriterHandler which appends to the file at the given
// path, creating it if necessary.
// templateString must be a template supported by the sawmill/event/formatter package.
// If the templateString is empty, the WriterHandler will use sawmill/event/formatter.SIMPLE_FORMAT.
func Append(path string, mode os.FileMode, templateString string) (*WriterHandler, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, mode)
	if err != nil {
		return nil, err
	}

	return New(f, templateString)
}

// Event accepts an event, formats it, and writes it to the WriterHandler's Output.
func (handler *WriterHandler) Event(logEvent *event.Event) error {
	//handler.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))
	var templateBuffer bytes.Buffer
	handler.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
	templateBuffer.WriteByte('\n')
	handler.Output.Write(templateBuffer.Bytes())

	return nil
}
