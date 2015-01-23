package handler

import (
  "github.com/phemmer/sawmill/event"
  "github.com/phemmer/sawmill/event/formatter"
	"golang.org/x/crypto/ssh/terminal"
	"text/template"
  "io"
	"bytes"
	"fmt"
)

type Handler interface {
	Event(event *event.Event) error
}

func IsTerminal(stream interface{Fd() uintptr}) bool {
  return terminal.IsTerminal(int(stream.Fd()))
}

type EventIOWriter struct {
	Output io.Writer
	Template *template.Template
}
func NewEventIOWriter (output io.Writer, templateString string) (*EventIOWriter, error) {
	if templateString == "" {
		templateString = formatter.SIMPLE_FORMAT
	}
	formatterTemplate, err := template.New("").Parse(templateString)
	if err != nil {
		fmt.Printf("Error parsing template: %s", err) //TODO send message somewhere else?
		return nil, err
	}
	ewriter := &EventIOWriter{
		Output: output,
		Template: formatterTemplate,
	}
	return ewriter, nil
}
func (ewriter *EventIOWriter) Event(logEvent *event.Event) (error) {
	//ewriter.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))
	var templateBuffer bytes.Buffer
	ewriter.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
	templateBuffer.WriteByte('\n')
	ewriter.Output.Write(templateBuffer.Bytes())

	return nil
}
