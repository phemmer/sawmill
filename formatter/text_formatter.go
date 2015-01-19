package formatter

import (
  "github.com/phemmer/sawmill/event"
  "golang.org/x/crypto/ssh/terminal"
  "fmt"
  "reflect"
  "sort"
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

type TextFormatter struct {
  UseColor bool
	DoAlignment bool
	TimeFormat string
}
func NewTextFormatter(color bool) *TextFormatter {
  return &TextFormatter{
		UseColor: color,
		DoAlignment: true,
		TimeFormat: "2006-01-02_15:04:05.000",
	}
}
func (formatter *TextFormatter) Format(logEvent *event.Event) ([]byte) {
	timestamp := formatter.FormatTimestamp(logEvent)
	level := formatter.FormatLevel(logEvent)
	message := formatter.FormatMessage(logEvent)
	fields := formatter.FormatFields(logEvent)

  buf := []byte(fmt.Sprintf("%s %s %s %s", timestamp, level, message, fields))

	return buf
}

func (formatter *TextFormatter) FormatTimestamp(logEvent *event.Event) string {
	return logEvent.Timestamp.Format(formatter.TimeFormat)
}
func (formatter *TextFormatter) FormatLevel(logEvent *event.Event) string {
	var levelName string
	if formatter.UseColor {
		var levelColor []byte
		if logEvent.Level <= event.Error {
			levelColor = colors.Red
		} else if logEvent.Level == event.Warning {
			levelColor = colors.Yellow
		} else {
			levelColor = colors.Blue
		}
		levelName = fmt.Sprintf("%s%s%s", levelColor, logEvent.LevelName(), colors.Reset)
	} else {
		levelName = logEvent.LevelName()
	}

	padding := []byte{}
	if formatter.DoAlignment {
		for i := len(logEvent.LevelName()); i < len(event.LevelName(event.Emergency)); i++ {
			padding = append(padding, ' ')
		}
	}

	return fmt.Sprintf("%s%s", levelName, padding)
}
func (formatter *TextFormatter) FormatMessage(logEvent *event.Event) string {
	if formatter.DoAlignment {
		return fmt.Sprintf("%-30s", logEvent.Message)
	}
		return logEvent.Message
}
func (formatter *TextFormatter) FormatFields(logEvent *event.Event) string {
  flatFields := flatten(logEvent.Fields)

  keys := make([]string, len(flatFields))
  i := 0
  for k := range flatFields {
    keys[i] = k
    i += 1
  }
  sort.Strings(keys)

	buf := []byte{}

  for _, k := range keys {
		if len(buf) > 0 {
			buf = append(buf, ' ')
		}
    v := flatFields[k]

		if formatter.UseColor {
			k = fmt.Sprintf("%s%s%s", colors.Cyan, k, colors.Reset)
		}
    buf = append(buf, []byte(fmt.Sprintf("%s=%v", k, v))...)
  }

  return string(buf)
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
