package event

import (
	"fmt"
	"reflect"
	"strconv"
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

	dataCopy, flatFields := deStruct(data)
	if flatFields == nil {
		flatFields = map[string]interface{}{"_": dataCopy}
	}

	event := &Event{
		Id:         id,
		Time:       now,
		Level:      level,
		Message:    message,
		Fields:     dataCopy,
		FlatFields: flatFields,
	}

	return event
}

func (event *Event) LevelName() string {
	return LevelName(event.Level)
}

// deStruct will take any input object and return a copy of it, and a `map[string]interface{}` of any nested attributes.
// If the object satisfies the `fmt.Stringer` interface (it has a `Strin()` method), then we will return that value without diving into nested attributes.
func deStruct(data interface{}) (interface{}, map[string]interface{}) {
	dataValue := reflect.ValueOf(data)
	return deStructValue(dataValue)
}
func deStructValue(dataValue reflect.Value) (interface{}, map[string]interface{}) {
	if stringer, ok := dataValue.Interface().(fmt.Stringer); ok {
		newData := stringer.String()
		return newData, nil
	}

	kind := dataValue.Kind()
	switch kind {
	case reflect.Ptr, reflect.Interface:
		return deStructReference(dataValue)
	case reflect.Struct:
		return deStructStruct(dataValue)
	case reflect.Map:
		return deStructMap(dataValue)
	case reflect.Array, reflect.Slice:
		return deStructSlice(dataValue)
	default:
		return deStructScalar(dataValue)
	}
}
func deStructReference(dataValue reflect.Value) (interface{}, map[string]interface{}) {
	return deStructValue(dataValue.Elem())
}
func deStructStruct(dataValue reflect.Value) (interface{}, map[string]interface{}) {
	newData := make(map[string]interface{})
	flatData := make(map[string]interface{})

	structType := reflect.TypeOf(dataValue.Interface())
	for i := 0; i < dataValue.NumField(); i++ {
		subDataValue := dataValue.Field(i)
		if !subDataValue.CanInterface() { // skip if it's unexported
			continue
		}

		key := structType.Field(i).Name

		fieldValue, fieldMap := deStructValue(subDataValue)
		newData[key] = fieldValue

		if fieldMap == nil { // non-nested value (scalar or byte slice)
			flatData[key] = fieldValue
		} else {
			for fieldMapKey, fieldMapValue := range fieldMap {
				flatKey := key + "." + fieldMapKey
				flatData[flatKey] = fieldMapValue
			}
		}
	}

	return newData, flatData
}
func deStructMap(dataValue reflect.Value) (interface{}, map[string]interface{}) {
	newData := make(map[interface{}]interface{})
	flatData := make(map[string]interface{})

	for _, keyValue := range dataValue.MapKeys() {
		subDataValue := dataValue.MapIndex(keyValue)
		keyInterface, _ := deStructValue(keyValue) // TODO just use `fmt.Sprintf("%v", keyValue)`?
		key := fmt.Sprintf("%v", keyInterface)

		fieldValue, fieldMap := deStructValue(subDataValue)
		newData[keyInterface] = fieldValue

		if fieldMap == nil { // non-nested value (scalar or byte slice)
			flatData[key] = fieldValue
		} else {
			for fieldMapKey, fieldMapValue := range fieldMap {
				flatKey := key + "." + fieldMapKey
				flatData[flatKey] = fieldMapValue
			}
		}
	}

	return newData, flatData
}
func deStructSlice(dataValue reflect.Value) (interface{}, map[string]interface{}) {
	if dataValue.Kind() == reflect.Uint8 {
		newDataValue := reflect.MakeSlice(dataValue.Type(), dataValue.Len(), dataValue.Cap())
		newDataValue = reflect.AppendSlice(newDataValue, dataValue)
		return newDataValue.Interface(), nil
	}

	newData := make([]interface{}, dataValue.Len())
	flatData := make(map[string]interface{})
	for i := 0; i < dataValue.Len(); i++ {
		subDataValue := dataValue.Index(i)
		key := strconv.Itoa(i)

		fieldValue, fieldMap := deStructValue(subDataValue)
		newData[i] = fieldValue

		if fieldMap == nil { // non-nested value (scalar or byte slice)
			flatData[key] = fieldValue
		} else {
			for fieldMapKey, fieldMapValue := range fieldMap {
				flatKey := key + "." + fieldMapKey
				flatData[flatKey] = fieldMapValue
			}
		}
	}

	return newData, flatData
}
func deStructScalar(dataValue reflect.Value) (interface{}, map[string]interface{}) {
	var newData interface{}
	if dataValue.IsValid() {
		newData = dataValue.Interface()
	} else {
		newData = nil
	}

	return newData, nil
}
