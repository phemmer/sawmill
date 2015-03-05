package event

import (
	"fmt"
	"reflect"
	"time"
)

type Level int

const (
	Debug, Dbg Level = iota, iota
	Info, _
	Notice, _
	Warning, Warn
	Error, Err
	Critical, Crit
	Alert, Alrt
	Emergency, Emerg
)

var LevelNames = [8]string{
	"Debug",
	"Info",
	"Notice",
	"Warning",
	"Error",
	"Critical",
	"Alert",
	"Emergency",
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

//TODO break each kind up into separate functions
//TODO This is probably rather ineffecient. We should look into how the `fmt` package works and see what we can rip out of it.
//     Basically the end goal of this function is to have a single level map with keys and values (flatFields), and to copy the data in the original 'fields' struct so that there's no possible race conditions due to modifications after the event was generated.
func deStruct(data interface{}, parent string, flatFields map[string]interface{}) interface{} {
	dataValue := reflect.ValueOf(data)
	for dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	kind := dataValue.Kind()

	if kind == reflect.Struct {
		//TODO for simple types, such as time.Time, copy them into newData instead of stringifying
		if stringer, ok := dataValue.Interface().(fmt.Stringer); ok {
			newData := stringer.String()
			flatFields[parent] = newData
			return newData
		}

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

		if errorer, ok := dataValue.Interface().(error); ok {
			errString := errorer.Error()
			if len(newData) == 0 {
				// this was a struct with no exported attributes on it.
				// so just assume the thing is a pure error object, and return the error string
				flatFields[parent] = errString
				return errString
			}
			if errString != "" {
				// this is a struct satisfying the error interface, but it has exported attributes as well
				// set the 'Error' field as if it were just a regular attribute
				key := parent + ".Error"
				flatFields[key] = errString
				newData["Error"] = errString
			}
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
		if byteSlice, ok := dataValue.Interface().([]byte); ok {
			var newData []byte
			newData = append(newData, byteSlice...)
			flatFields[parent] = newData
			return newData
		}

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
