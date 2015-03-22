/*
The writer package is an event handler responsible for sending events to a generic IO writer.
*/
package writer

import (
	"bytes"
	"fmt"
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"text/template"
)

// IsTerminal returns whether the given stream (File) is attached to a TTY.
func IsTerminal(stream interface {
	Fd() uintptr
}) bool {
	return terminal.IsTerminal(int(stream.Fd()))
}

// EventWriter is responsible for converting an event into text using a template, and then sending that text to an io.Writer.
type EventWriter struct {
	Output   io.Writer
	Template *template.Template
}

// NewEventWriter constructs a new EventWriter.
// templateString must be a template supported by the sawmill/event/formatter package.
// If the templateString is empty, the EventWriter will use sawmill/event/formatter.SIMPLE_FORMAT.
func NewEventWriter(output io.Writer, templateString string) (*EventWriter, error) {
	if templateString == "" {
		templateString = formatter.SIMPLE_FORMAT
	}
	formatterTemplate, err := template.New("").Parse(templateString)
	if err != nil {
		fmt.Printf("Error parsing template: %s", err) //TODO send message somewhere else?
		return nil, err
	}
	ewriter := &EventWriter{
		Output:   output,
		Template: formatterTemplate,
	}
	return ewriter, nil
}

// Event accepts an event, formats it, and writes it to the EventWriter's Output.
func (ewriter *EventWriter) Event(logEvent *event.Event) error {
	//ewriter.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))
	var templateBuffer bytes.Buffer
	ewriter.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
	templateBuffer.WriteByte('\n')
	ewriter.Output.Write(templateBuffer.Bytes())

	return nil
}
