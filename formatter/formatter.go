package formatter

import (
  "github.com/phemmer/sawmill/event"
  "fmt"
)

type Formatter interface {
  Format(event *event.Event) []byte
}

type TextFormatter struct {
}
func NewTextFormatter() *TextFormatter {
  return &TextFormatter{}
}
func (formatter *TextFormatter) Format(event *event.Event) ([]byte) {
  return []byte(fmt.Sprintf("%#v\n", event))
}
