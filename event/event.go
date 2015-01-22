package event

import (
  "time"
	"reflect"
	"fmt"
)

type Level int
const (
	Emergency, Emerg Level = iota, iota
	Alert, Alrt
	Critical, Crit
	Error, Err
	Warning, Warn
	Notice, _
	Info, _
	Debug, Dbg
)
var LevelNames = [8]string{
	"Emergency",
	"Alert",
	"Critical",
	"Error",
	"Warning",
	"Notice",
	"Info",
	"Debug",
}
func LevelName(level Level) string {
	return LevelNames[level]
}

type Event struct {
	Level Level
	Time time.Time
	Message string
	Fields interface{}
}

func (event *Event) LevelName() string {
	return LevelName(event.Level)
}
func (event *Event) FlatFields() map[string]interface{} {
	return flatten(event.Fields)
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
