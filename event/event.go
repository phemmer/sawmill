package event

import (
	//"reflect"
  "time"
	//"github.com/fatih/structs"
	//"code.google.com/p/rog-go/exp/deepcopy"
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

type Event struct {
	Level Level
	Timestamp time.Time
	Message string
	Fields interface{}
}

func (event *Event) FieldsMap (map[string]interface{}) {
	/*
	fieldsV := reflect.ValueOf(event.Fields)
	if fieldsV.Kind() == reflect.Ptr {
		fieldsV = fieldsV.Elem()
	}

	kind := fieldsV.Kind()
	if kind == reflect.Struct {
		fieldsMap := structs.Map(fieldsV.Interface())
	} else if kind == reflect.Array || kind == reflect.Slice {
		fieldsMap := map[string]interface{}
		i := 0
		for value := range event.Fields {
			fieldsMap[sprintf("%d", i)] = event.Fields[i]
			i += 1
		}
	} else if kind == reflect.Map {
	} else {
	}
	*/
}
func (event *Event) FieldsMapFlattened (map[string]string) {
}
