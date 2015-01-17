package hook

import (
  "github.com/phemmer/sawmill/event"
  "github.com/phemmer/sawmill/formatter"
  "io"
)

type Hook interface {
	Event(event *event.Event) error
}

type HookIOWriter struct {
	Output io.Writer
  Formatter formatter.Formatter
}
func NewHookIOWriter (output io.Writer, formatter formatter.Formatter) (*HookIOWriter) {
  return &HookIOWriter{Output: output, Formatter: formatter}
}
func (hook *HookIOWriter) Event(event *event.Event) (error) {
	//hook.Output.Write([]byte(fmt.Sprintf("%#v\n", event)))
	buf := append(hook.Formatter.Format(event), '\n')
	hook.Output.Write(buf)

	return nil
}
