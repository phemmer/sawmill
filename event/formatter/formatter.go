// The formatter package is used to wrap an event with helper functions so it can be easily used in text templates.
package formatter

import (
	"fmt"
	"github.com/phemmer/sawmill/event"
	"strconv"
	"strings"
	"unicode"
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

	Underline: []byte{27, '[', '4', 'm'},

	Reset: []byte{27, '[', '0', 'm'},
}

const (
	SIMPLE_FORMAT          = "{{.Message}} --{{range $k,$v := .Fields}} {{$k}}={{$.Quote $v}}{{end}}"
	CONSOLE_COLOR_FORMAT   = "{{.Time \"2006-01-02_15:04:05.000\"}} {{.Level | .Color | printf \"%s>\" | .Pad -10}} {{.Message | .Pad -30}}{{range $k,$v := .Fields}} {{$k | $.Color}}={{$.Quote $v}}{{end}}"
	CONSOLE_NOCOLOR_FORMAT = "{{.Time \"2006-01-02_15:04:05.000\"}} {{.Level | printf \"%s>\" | .Pad -10}} {{.Message | .Pad -30}}{{range $k,$v := .Fields}} {{$k}}={{$.Quote $v}}{{end}}"
)

type Formatter struct { // TODO(.) it feels really weird not having the formatter contain the format.
	Event *event.Event
}

// EventFormatter constructs a new Formatter containing the given event.
func EventFormatter(logEvent *event.Event) *Formatter {
	return &Formatter{Event: logEvent}
}

// Time converts the event's time into a string according to the given format.
// The format is the same as used by time.Format().
func (formatter *Formatter) Time(format string) string {
	return formatter.Event.Time.Format(format)
}

// Level converts the event's level into a string.
func (formatter *Formatter) Level() string {
	return formatter.Event.Level.String()
}

// Color wraps the given text in ANSI color escapes appropriate to the event's level.
// Error and higher are red. Warning is yellow. Notice and lower are cyan.
func (formatter *Formatter) Color(text string) string {
	var levelColor []byte
	if formatter.Event.Level >= event.Error {
		levelColor = colors.Red
	} else if formatter.Event.Level == event.Warning {
		levelColor = colors.Yellow
	} else {
		levelColor = colors.Cyan
	}
	return fmt.Sprintf("%s%s%s", levelColor, text, colors.Reset)
}

// ToString converts any arbitrary data into a string.
func (formatter *Formatter) ToString(data interface{}) string {
	if str, ok := data.(string); ok {
		return str
	}

	if byteSlice, ok := data.([]byte); ok {
		return string(byteSlice)
	}

	return fmt.Sprintf("%v", data)
}

func needQuote(str string) bool {
	for _, char := range str {
		if unicode.IsSpace(char) || !unicode.IsPrint(char) || unicode.Is(unicode.Quotation_Mark, char) {
			return true
		}
	}

	return false
}

// Quote converts the given data into a string, and adds quotes if necessary.
// Quotes are deemed necessary if the string contains whitespace, non-printable characters, or quotation marks.
func (formatter *Formatter) Quote(data interface{}) string {
	str := formatter.ToString(data)

	if needQuote(str) {
		return strconv.Quote(str)
	}

	return str
}

// Pad pads the provided text to the specified length, while properly handling the color escape codes.
// Like the `%-10s` format, negative values mean pad on the right, where as positive values mean pad on the left.
func (formatter *Formatter) Pad(size int, text string) string {
	pos := 0
	colorLen := 0
	for index := strings.Index(text[pos:], "["); index != -1; index = strings.Index(text[pos:], "[") {
		pos = pos + index
		index = strings.Index(text[pos:], "m")
		if index == -1 {
			break
		}
		colorLen = colorLen + index + 1 // + 1 because 'index' is effectively the number of characters before 'm', where we want length including 'm'
		pos = pos + index + 1
	}
	textLen := len(text) - colorLen

	if size < 0 {
		padLen := -size - textLen
		if padLen > 0 {
			return fmt.Sprintf("%s%s", text, strings.Repeat(" ", padLen))
		}
	} else {
		padLen := size - textLen
		if padLen > 0 {
			return fmt.Sprintf("%s%s", strings.Repeat(" ", padLen), text)
		}
	}

	return text
}

// Message returns the event's message.
func (formatter *Formatter) Message() string {
	return formatter.Event.Message
}

// Fields returns a map of the event's fields in single-level dot key notation.
// If the event's fields contain nested data, the nested data is flattened into a single level, with the previously nested levels added as keys prefixed with the parent's level, and a dot ('.').
//
// For example:
//  map[string]interface{}{"foo": map[string]interface{}{"bar": "baz"}}
// Will convert to `foo.bar=baz`
func (formatter *Formatter) Fields() map[string]interface{} {
	return formatter.Event.FlatFields
}
