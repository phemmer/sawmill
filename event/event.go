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
  FlatFields map[string]interface{}
}

func NewEvent(level Level, message string, data interface{}) *Event {
  now := time.Now()

  // A
  //fields, flatFields := fieldWalk(data)

  // B
  flatFields := map[string]interface{}{}
  fields := deStruct(data, "", flatFields)

  event := &Event{
    Time: now,
    Level: level,
    Message: message,
    Fields: fields,
    FlatFields: flatFields,
  }

  return event
}

func (event *Event) LevelName() string {
	return LevelName(event.Level)
}

func fieldWalk(data interface{}) (interface{}, map[string]interface{}) {
  data, flatFields, isScalar := fieldWalkWith(data)
  if isScalar {
    data = map[string]interface{}{"_": data}
  }

  return data, flatFields
}
func fieldWalkWith(data interface{}) (interface{}, map[string]interface{}, bool) {
  dataValue := reflect.ValueOf(data)
  for dataValue.Kind() == reflect.Ptr {
    dataValue = dataValue.Elem()
  }

  var result interface{}
  flatFields := map[string]interface{}{}
  isScalar := false

  kind := dataValue.Kind()
  if kind == reflect.Struct {
    realResult := make(map[string]interface{})
		structType := reflect.TypeOf(dataValue.Interface())
		for i := 0; i < dataValue.NumField(); i++ {
			field := dataValue.Field(i)
			if ! field.CanInterface() { // skip if it's unexported
				continue
			}
			key := structType.Field(i).Name

      subData, subFlatFields, subIsScalar := fieldWalkWith(field.Interface())
      realResult[key] = subData
      if subIsScalar {
        flatFields[key] = subData
      } else {
        for subFlatKey,subFlatData := range subFlatFields {
          flatKey := fmt.Sprintf("%s.%v", key, subFlatKey)
          flatFields[flatKey] = subFlatData
        }
      }
		}
    result = realResult
  } else if kind == reflect.Map {
    realResult := map[interface{}]interface{}{}
    for _, keyValue := range dataValue.MapKeys() {
      key := keyValue.Interface()
      subDataValue := dataValue.MapIndex(keyValue)

      subData, subFlatFields, subIsScalar := fieldWalkWith(subDataValue.Interface())
      realResult[key] = subData
      if subIsScalar {
        flatFields[fmt.Sprintf("%v", key)] = subData
      } else {
        for subFlatKey,subFlatData := range subFlatFields {
          flatKey := fmt.Sprintf("%v.%v", key, subFlatKey)
          flatFields[flatKey] = subFlatData
        }
      }
    }
    result = realResult
  } else if kind == reflect.Array || kind == reflect.Slice {
    realResult := make([]interface{}, dataValue.Len())
    for i := 0; i < dataValue.Len(); i++ {
      key := i
      subData := dataValue.Index(i).Interface()
      subData, subFlatFields, subIsScalar := fieldWalkWith(subData)
      if subIsScalar {
        flatFields[fmt.Sprintf("%d", key)] = subData
      } else {
        for subFlatKey,subFlatData := range subFlatFields {
          flatKey := fmt.Sprintf("%d.%v", key, subFlatKey)
          flatFields[flatKey] = subFlatData
        }
      }
    }
    result = realResult
  } else {
    result = dataValue.Interface()
    flatFields["_"] = result
    isScalar = true
  }

  return result, flatFields, isScalar
}


func deStruct(data interface{}, parent string, flatFields map[string]interface{}) (interface{}) {
	dataValue := reflect.ValueOf(data)
	for dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

  kind := dataValue.Kind()

	if kind == reflect.Struct {
		result := make(map[string]interface{})
		structType := reflect.TypeOf(dataValue.Interface())
		for i := 0; i < dataValue.NumField(); i++ {
			field := dataValue.Field(i)
			if ! field.CanInterface() { // skip if it's unexported
				continue
			}
			k := structType.Field(i).Name

      var kFlat string
      if parent == "" {
        kFlat = k
      } else {
        kFlat = fmt.Sprintf("%s.%s", parent, k)
      }

			result[k] = deStruct(field.Interface(), kFlat, flatFields)
		}
		return result
	} else if dataValue.Kind() == reflect.Map {
		result := make(map[interface{}]interface{})
		for _, kValue := range dataValue.MapKeys() {
			vValue := dataValue.MapIndex(kValue)
			k := deStruct(kValue.Interface(), "", nil)

      var kFlat string
      if parent == "" {
        kFlat = fmt.Sprintf("%v", k)
      } else {
        kFlat = fmt.Sprintf("%s.%v", parent, k)
      }

			result[k] = deStruct(vValue.Interface(), kFlat, flatFields)
		}
		return result
	} else if dataValue.Kind() == reflect.Array || dataValue.Kind() == reflect.Slice {
		var result []interface{}
    for i := 0; i < dataValue.Len(); i++ {
      v := dataValue.Index(i).Interface()
      var kFlat string
      if parent == "" {
        kFlat = fmt.Sprintf("%d", i)
      } else {
        kFlat = fmt.Sprintf("%s.%d", parent, i)
      }

      v = deStruct(v, kFlat, flatFields)
			result = append(result, v)
		}
		return result
	}
	// scalar
  if flatFields != nil {
    if parent == "" {
      parent = "."
    }
    flatFields[parent] = dataValue.Interface()
  }
	return dataValue.Interface()
}
