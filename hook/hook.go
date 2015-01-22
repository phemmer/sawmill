package hook

import (
  "github.com/phemmer/sawmill/event"
  "github.com/phemmer/sawmill/event/formatter"
	"golang.org/x/crypto/ssh/terminal"
	"text/template"
  "io"
	"bytes"
	"fmt"
)

type Hook interface {
	Event(event *event.Event) error
}

func IsTerminal(stream interface{Fd() uintptr}) bool {
  return terminal.IsTerminal(int(stream.Fd()))
}

type HookIOWriter struct {
	Output io.Writer
	Template *template.Template
}
func NewHookIOWriter (output io.Writer, templateString string) (*HookIOWriter, error) {
	if templateString == "" {
		templateString = formatter.SIMPLE_FORMAT
	}
	formatterTemplate, err := template.New("").Parse(templateString)
	if err != nil {
		fmt.Printf("Error parsing template: %s", err) //TODO send message somewhere else?
		return nil, err
	}
	hook := &HookIOWriter{
		Output: output,
		Template: formatterTemplate,
	}
	return hook, nil
}
func (hook *HookIOWriter) Event(logEvent *event.Event) (error) {
	//hook.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))
	var templateBuffer bytes.Buffer
	hook.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
	templateBuffer.WriteByte('\n')
	hook.Output.Write(templateBuffer.Bytes())

	return nil
}
