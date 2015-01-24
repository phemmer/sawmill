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

func IsTerminal(stream interface {
	Fd() uintptr
}) bool {
	return terminal.IsTerminal(int(stream.Fd()))
}

type EventWriter struct {
	Output   io.Writer
	Template *template.Template
}

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
func (ewriter *EventWriter) Event(logEvent *event.Event) error {
	//ewriter.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))
	var templateBuffer bytes.Buffer
	ewriter.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
	templateBuffer.WriteByte('\n')
	ewriter.Output.Write(templateBuffer.Bytes())

	return nil
}
