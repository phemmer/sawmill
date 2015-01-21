package formatter

import (
  "github.com/phemmer/sawmill/event"
  "golang.org/x/crypto/ssh/terminal"
  "fmt"
  "reflect"
  //"sort"
	"bytes"
	"text/template"
)

func IsTerminal(stream interface{Fd() uintptr}) bool {
  return terminal.IsTerminal(int(stream.Fd()))
}

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

type TextFormatter struct {
	Template *template.Template
}
func NewTextFormatter(templateString string) *TextFormatter {
	if templateString == "" {
		templateString = SIMPLE_FORMAT
	}
	formatterTemplate, err := template.New("").Parse(templateString)
	if err != nil {
		fmt.Printf("Error parsing template: %s", err)
		//TODO proper error handler
	}
	textFormatter := &TextFormatter{
		Template: formatterTemplate,
	}
	return textFormatter
}
func (formatter *TextFormatter) Format(logEvent *event.Event) ([]byte) {
	var templateBuffer bytes.Buffer
	flatFields := flatten(logEvent.Fields)
	templateContext := &TemplateContext{Event: logEvent, Fields: flatFields}
	err := formatter.Template.Execute(&templateBuffer, templateContext)
	if err != nil {
		fmt.Printf("Error executing template: %s\n", err)
		//TODO proper error handler
	}
	return templateBuffer.Bytes()
}

type TemplateContext struct {
	Event *event.Event
	Fields map[string]interface{}
}
func (tc *TemplateContext) Time(format string) string {
	return tc.Event.Timestamp.Format(format)
}
func (tc *TemplateContext) Level() string {
	return tc.Event.LevelName()
}
func (tc *TemplateContext) Color(text string) string {
	var levelColor []byte
	if tc.Event.Level <= event.Error {
		levelColor = colors.Red
	} else if tc.Event.Level == event.Warning {
		levelColor = colors.Yellow
	} else {
		levelColor = colors.Blue
	}
	return fmt.Sprintf("%s%s%s", levelColor, text, colors.Reset)
}
func (tc *TemplateContext) Pad(size int, text string) string {
	return text //TODO
}
func (tc *TemplateContext) Message() string {
	return tc.Event.Message
}

func flatten(fields interface{}) (map[string]interface{}) {
  flat := make(map[string]interface{})

  value := reflect.ValueOf(fields)
  for value.Kind() == reflect.Ptr { // shouldn't ever happen since deStruct() also does this
    value = value.Elem()
  }

  //fmt.Printf("flattening: %#v\n", fields)
  if value.Kind() == reflect.Map {
    for _, kV := range value.MapKeys() {
      vV := value.MapIndex(kV)
      k := kV.Interface()
      flattenValue(flat, k, vV)
    }
  } else if value.Kind() == reflect.Array || value.Kind() == reflect.Slice {
    for k, v := range value.Interface().([]interface{}) {
      flattenValue(flat, k, reflect.ValueOf(v))
    }
  } else {
    if value.IsValid() {
      flat[""] = value.Interface()
    } else {
      flat[""] = nil
    }
  }

  //fmt.Printf("Flatten result: %#v\n", flat)
  return flat
}
func flattenValue(flattened map[string]interface{}, parentKey interface{}, value reflect.Value) {
  kind := value.Kind()
  for kind == reflect.Ptr || kind == reflect.Interface {
    kind = value.Elem().Kind()
  }

  if kind == reflect.Struct || kind == reflect.Map || kind == reflect.Array || kind == reflect.Slice {
    for vk,vv := range flatten(value.Interface()) {
      flat_k := fmt.Sprintf("%s.%s", parentKey, vk)
      flattened[flat_k] = vv
    }
  } else {
    flat_k := fmt.Sprintf("%s", parentKey)
    flattened[flat_k] = value.Interface()
  }
}
