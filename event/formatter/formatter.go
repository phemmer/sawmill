package formatter

import (
  "github.com/phemmer/sawmill/event"
  "fmt"
)

type ansiColorCodes struct {
	Bold, Normal, Black, BlackBold, Red, RedBold, Green, GreenBold, Yellow, YellowBold, Blue, BlueBold, Magenta, MagentaBold, Cyan, CyanBold, White, WhiteBold, Underline, Reset []byte
}
var colors = ansiColorCodes{
	Bold:        []byte{27, '[', '1', 'm'},
	Normal:      []byte{27, '[', '2', '2', 'm'},
	Black:       []byte{27, '[', '3', '0', 'm'},
	BlackBold:   []byte{27, '[', '3', '0', ';', '1', 'm'},
	Red:         []byte{27, '[', '3', '1', 'm'},
	RedBold:     []byte{27, '[', '3', '1', ';', '1', 'm'},
	Green:       []byte{27, '[', '3', '2', 'm'},
	GreenBold:   []byte{27, '[', '3', '2', ';', '1', 'm'},
	Yellow:      []byte{27, '[', '3', '3', 'm'},
	YellowBold:  []byte{27, '[', '3', '3', ';', '1', 'm'},
	Blue:        []byte{27, '[', '3', '4', 'm'},
	BlueBold:    []byte{27, '[', '3', '4', ';', '1', 'm'},
	Magenta:     []byte{27, '[', '3', '5', 'm'},
	MagentaBold: []byte{27, '[', '3', '5', ';', '1', 'm'},
	Cyan:        []byte{27, '[', '3', '6', 'm'},
	CyanBold:    []byte{27, '[', '3', '6', ';', '1', 'm'},
	White:       []byte{27, '[', '3', '7', 'm'},
	WhiteBold:   []byte{27, '[', '3', '7', ';', '1', 'm'},

	Underline:   []byte{27, '[', '4', 'm'},

	Reset:   []byte{27, '[', '0', 'm'},
}

const (
	SIMPLE_FORMAT = "{{.Message}}{{range $k,$v := .Fields}} {{$k}}={{$v}}{{end}}"
	CONSOLE_COLOR_FORMAT = "{{.Time \"2006-01-02_15:04:05.000\"}} {{.Level | .Color | printf \"%s>\" | .Pad -9}} {{.Message | .Pad -30}}{{range $k,$v := .Fields}} {{$k | $.Color}}={{$v}}{{end}}"
	CONSOLE_NOCOLOR_FORMAT = "{{.Time \"2006-01-02_15:04:05.000\"}} {{.Level | printf \"%s>\" | .Pad -9}} {{.Message | .Pad -30}}{{range $k,$v := .Fields}} {{$k}}={{$v}}{{end}}"
)

type Formatter struct {
	Event *event.Event
}
func EventFormatter(logEvent *event.Event) (*Formatter) {
  return &Formatter{Event: logEvent}
}
func (formatter *Formatter) Time(format string) string {
	return formatter.Event.Time.Format(format)
}
func (formatter *Formatter) Level() string {
	return formatter.Event.LevelName()
}
func (formatter *Formatter) Color(text string) string {
	var levelColor []byte
	if formatter.Event.Level <= event.Error {
		levelColor = colors.Red
	} else if formatter.Event.Level == event.Warning {
		levelColor = colors.Yellow
	} else {
		levelColor = colors.Blue
	}
	return fmt.Sprintf("%s%s%s", levelColor, text, colors.Reset)
}
func (formatter *Formatter) Pad(size int, text string) string {
	return text //TODO
}
func (formatter *Formatter) Message() string {
	return formatter.Event.Message
}
func (formatter *Formatter) Fields() map[string]interface{} {
	return formatter.Event.FlatFields()
}
