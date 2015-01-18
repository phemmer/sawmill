package formatter

import (
  "github.com/phemmer/sawmill/event"
  "fmt"
  "reflect"
  "sort"
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
  fields := flatten(event.Fields)

  buf := []byte(fmt.Sprintf("%s> %s ", event.LevelName(), event.Message))

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
