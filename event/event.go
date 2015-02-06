package event

import (
	"fmt"
	"reflect"
	"time"
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
	Id         uint64
	Level      Level
	Time       time.Time
	Message    string
	Fields     interface{}
	FlatFields map[string]interface{}
}

func NewEvent(id uint64, level Level, message string, data interface{}) *Event {
	now := time.Now()

	var fields interface{}
	flatFields := map[string]interface{}{}
	if data != nil {
		fields = deStruct(data, "", flatFields)
	}

	event := &Event{
		Id:         id,
		Time:       now,
		Level:      level,
		Message:    message,
		Fields:     fields,
		FlatFields: flatFields,
	}

	return event
}

func (event *Event) LevelName() string {
	return LevelName(event.Level)
}

func deStruct(data interface{}, parent string, flatFields map[string]interface{}) interface{} {
	dataValue := reflect.ValueOf(data)
	for dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	kind := dataValue.Kind()

	if kind == reflect.Struct {
		newData := make(map[string]interface{})
		structType := reflect.TypeOf(dataValue.Interface())
		for i := 0; i < dataValue.NumField(); i++ {
			subDataValue := dataValue.Field(i)
			if !subDataValue.CanInterface() { // skip if it's unexported
				continue
			}
			key := structType.Field(i).Name

			var keyFlat string
			if parent == "" {
				keyFlat = key
			} else {
				keyFlat = fmt.Sprintf("%s.%s", parent, key)
			}

			newData[key] = deStruct(subDataValue.Interface(), keyFlat, flatFields)
		}
		return newData
	} else if dataValue.Kind() == reflect.Map {
		newData := make(map[interface{}]interface{})
		for _, keyValue := range dataValue.MapKeys() {
			subDataValue := dataValue.MapIndex(keyValue)
			key := deStruct(keyValue.Interface(), "", nil)

			var keyFlat string
			if parent == "" {
				keyFlat = fmt.Sprintf("%v", key)
			} else {
				keyFlat = fmt.Sprintf("%s.%v", parent, key)
			}

			newData[key] = deStruct(subDataValue.Interface(), keyFlat, flatFields)
		}
		return newData
	} else if dataValue.Kind() == reflect.Array || dataValue.Kind() == reflect.Slice {
		var newData []interface{}

		for i := 0; i < dataValue.Len(); i++ {
			subData := dataValue.Index(i).Interface()
			var keyFlat string
			if parent == "" {
				keyFlat = fmt.Sprintf("%d", i)
			} else {
				keyFlat = fmt.Sprintf("%s.%d", parent, i)
			}

			subData = deStruct(subData, keyFlat, flatFields)
			newData = append(newData, subData)
		}
		return newData
	}
	// scalar
	var newData interface{}
	if dataValue.IsValid() {
		newData = dataValue.Interface()
	} else {
		newData = nil
	}
	if flatFields != nil {
		if parent == "" {
			parent = "."
		}
		flatFields[parent] = newData
	}
	return newData
}
