package formatter

import (
  "github.com/phemmer/sawmill/event"
)

type Formatter interface {
  Format(event *event.Event) []byte
}
