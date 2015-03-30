/*
The writer package is an event handler responsible for sending events to a generic IO writer.
*/
package writer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/phemmer/sawmill/event"
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
// templateString must be a template supported by the sawmill/event package.
// If the templateString is empty, the WriterHandler will use sawmill/event.SimpleFormat.
func New(output io.Writer, templateString string) (*WriterHandler, error) {
	if templateString == "" {
		templateString = event.SimpleFormat
	}
	formatterTemplate, err := event.NewTemplate("", templateString)
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

// Event accepts an event, formats it, and writes it to the WriterHandler's Output.
func (handler *WriterHandler) Event(logEvent *event.Event) error {
	//handler.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))
	var templateBuffer bytes.Buffer
	err := handler.Template.Execute(&templateBuffer, logEvent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return err
	}
	templateBuffer.WriteByte('\n')
	handler.Output.Write(templateBuffer.Bytes())

	return nil
}
