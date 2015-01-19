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
var colors = terminal.EscapeCodes{
	Black:   []byte{27, '[', '3', '0', 'm'},
	Red:     []byte{27, '[', '3', '1', 'm'},
	Green:   []byte{27, '[', '3', '2', 'm'},
	Yellow:  []byte{27, '[', '3', '3', 'm'},
	Blue:    []byte{27, '[', '3', '4', 'm'},
	Magenta: []byte{27, '[', '3', '5', 'm'},
	Cyan:    []byte{27, '[', '3', '6', 'm'},
	White:   []byte{27, '[', '3', '7', 'm'},

	Reset:   []byte{27, '[', '0', 'm'},
}

type TextFormatter struct {
  color bool
}
func NewTextFormatter(color bool) *TextFormatter {
  return &TextFormatter{color: color}
}
func (formatter *TextFormatter) Format(logEvent *event.Event) ([]byte) {
  fields := flatten(logEvent.Fields)

  timestamp := logEvent.Timestamp.Format("2006-01-02_15:04:05.00")
	var levelName string
	if formatter.color {
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
  buf := []byte(fmt.Sprintf("%s %s> %s ", timestamp, levelName, logEvent.Message))

  flatFields := flatten(fields)
  keys := make([]string, len(flatFields))
  i := 0
  for k := range flatFields {
    keys[i] = k
    i += 1
  }
  sort.Strings(keys)
  for _, k := range keys {
    v := flatFields[k]
    buf = append(buf, []byte(fmt.Sprintf("%s=%v ", k, v))...)
  }
  return buf
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
