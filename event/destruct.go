package event

import (
	"fmt"
	"reflect"
	"strconv"
)

// deStruct will take any input object and return a copy of it, a scalar representation, and a `map[string]interface{}` of any nested attributes.
// The scalar representation is so that if an object has an underlaying value, and then satisfies an interface, such as `Error()`, that we get both values.
// If the object satisfies the `fmt.Stringer` interface (it has a `String()` method), then we will return that value without diving into nested attributes.
func deStruct(data interface{}) (interface{}, interface{}, map[string]interface{}) {
	dataValue := reflect.ValueOf(data)
	return deStructValue(dataValue)
}
func deStructValue(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	var dataCopy interface{}
	var dataScalar interface{}
	var flatFields map[string]interface{}

	var deStructX func(reflect.Value) (interface{}, interface{}, map[string]interface{})
	kind := dataValue.Kind()
	switch kind {
	case reflect.Ptr:
		deStructX = deStructPointer
	case reflect.Interface:
		deStructX = deStructInterface
	case reflect.Struct:
		deStructX = deStructStruct
	case reflect.Map:
		deStructX = deStructMap
	case reflect.Array, reflect.Slice:
		deStructX = deStructSlice
	case reflect.Chan:
		deStructX = deStructChan
	case reflect.Func:
		deStructX = deStructFunction
	default:
		deStructX = deStructScalar
	}
	dataCopy, dataScalar, flatFields = deStructX(dataValue)

	if dataValue.IsValid() && dataValue.Kind() != reflect.Interface && !(dataValue.Kind() == reflect.Ptr && dataValue.IsNil()) {
		if stringer, ok := dataValue.Interface().(fmt.Stringer); ok {
			// has a string interface. Discard dataScalar and flatFields
			// Basically we called deStructX() only for the dataCopy
			dataScalar = stringer.String()
			flatFields = map[string]interface{}{}
		}

		if errorer, ok := dataValue.Interface().(error); ok {
			dataScalar = errorer.Error()
			flatFields = map[string]interface{}{}
		}
	}

	return dataCopy, dataScalar, flatFields
}
func deStructPointer(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	dataCopy, dataScalar, flatFields := deStructValue(dataValue.Elem())
	// this is since the original value was a pointer, for the copy we return a pointer as well
	// We can't just `return &dataCopy` as `dataCopy` is an `interface{}`, so this would return a pointer to an interface rather than a pointer to the copy itself
	dataCopyValue := reflect.ValueOf(dataCopy)
	var dataCopyPtr interface{}
	if dataCopyValue.IsValid() {
		dataCopyPtrValue := reflect.New(dataCopyValue.Type())
		dataCopyPtrValue.Elem().Set(dataCopyValue)
		dataCopyPtr = dataCopyPtrValue.Interface()
	} else {
		// copy is invalid (likely zero value)
		// return a zero-value of the same type as the original instead
		dataCopyPtr = reflect.Zero(dataValue.Type()).Interface()
	}
	return dataCopyPtr, dataScalar, flatFields
}
func deStructInterface(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	return deStructValue(dataValue.Elem())
}
func deStructStruct(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	newData := make(map[string]interface{})
	flatData := make(map[string]interface{})

	structType := reflect.TypeOf(dataValue.Interface())
	for i := 0; i < dataValue.NumField(); i++ {
		subDataValue := dataValue.Field(i)
		if !subDataValue.CanInterface() { // skip if it's unexported
			continue
		}

		key := structType.Field(i).Name

		fieldCopy, fieldScalar, fieldMap := deStructValue(subDataValue)
		newData[key] = fieldCopy

		if fieldScalar != nil {
			flatData[key] = fieldScalar
		}
		for fieldMapKey, fieldMapValue := range fieldMap {
			flatData[key+"."+fieldMapKey] = fieldMapValue
		}
	}

	return newData, nil, flatData
}
func deStructMap(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	newData := make(map[interface{}]interface{})
	flatData := make(map[string]interface{})

	for _, keyValue := range dataValue.MapKeys() {
		subDataValue := dataValue.MapIndex(keyValue)
		keyInterface, _, _ := deStructValue(keyValue) // TODO just use `fmt.Sprintf("%v", keyValue)`?
		key := fmt.Sprintf("%v", keyInterface)

		fieldCopy, fieldScalar, fieldMap := deStructValue(subDataValue)
		newData[keyInterface] = fieldCopy

		if fieldScalar != nil {
			flatData[key] = fieldScalar
		}
		for fieldMapKey, fieldMapValue := range fieldMap {
			flatData[key+"."+fieldMapKey] = fieldMapValue
		}
	}

	return newData, nil, flatData
}

func deStructSlice(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	if dataValue.Kind() == reflect.Uint8 {
		newDataValue := reflect.MakeSlice(dataValue.Type(), dataValue.Len(), dataValue.Cap())
		newDataValue = reflect.AppendSlice(newDataValue, dataValue)
		return newDataValue.Interface(), newDataValue.Interface(), nil
	}

	//TODO if the type inside the slice is not a struct, recreate the slice with the same definition
	newData := make([]interface{}, dataValue.Len())
	flatData := make(map[string]interface{})
	for i := 0; i < dataValue.Len(); i++ {
		subDataValue := dataValue.Index(i)
		key := strconv.Itoa(i)

		fieldCopy, fieldScalar, fieldMap := deStructValue(subDataValue)
		newData[i] = fieldCopy

		if fieldScalar != nil {
			flatData[key] = fieldScalar
		}
		for fieldMapKey, fieldMapValue := range fieldMap {
			flatData[key+"."+fieldMapKey] = fieldMapValue
		}
	}

	return newData, nil, flatData
}

func deStructChan(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	//return nil, fmt.Sprintf("%#v", dataValue.Interface()), map[string]interface{}{}
	return nil, nil, map[string]interface{}{}
}

func deStructFunction(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	//return nil, fmt.Sprintf("%#v", dataValue.Interface()), map[string]interface{}{}
	return nil, nil, map[string]interface{}{}
}

var scalarConversionMap = map[reflect.Kind]reflect.Type{
	reflect.Bool:       reflect.TypeOf(true),
	reflect.Float32:    reflect.TypeOf(float32(0)),
	reflect.Float64:    reflect.TypeOf(float64(0)),
	reflect.Complex64:  reflect.TypeOf(complex64(0)),
	reflect.Complex128: reflect.TypeOf(complex128(0)),
	reflect.Int:        reflect.TypeOf(int(0)),
	reflect.Int8:       reflect.TypeOf(int8(0)),
	reflect.Int16:      reflect.TypeOf(int16(0)),
	reflect.Int32:      reflect.TypeOf(int32(0)),
	reflect.Int64:      reflect.TypeOf(int64(0)),
	reflect.Uint:       reflect.TypeOf(uint(0)),
	reflect.Uint8:      reflect.TypeOf(uint8(0)),
	reflect.Uint16:     reflect.TypeOf(uint16(0)),
	reflect.Uint32:     reflect.TypeOf(uint32(0)),
	reflect.Uint64:     reflect.TypeOf(uint64(0)),
	reflect.Uintptr:    reflect.TypeOf(uintptr(0)),
	reflect.String:     reflect.TypeOf(string("")),
}

func deStructScalar(dataValue reflect.Value) (interface{}, interface{}, map[string]interface{}) {
	if !dataValue.IsValid() {
		return nil, nil, map[string]interface{}{}
	}

	var newData interface{}
	// here we convert the value to it's underlaying kind. This is so that if you do `type myint int`, we drop the `myint` and get a plain `int`. This is necessary so that during string conversion, we don't have any interface such as `Error()` or `String()` which might return different values.
	if convertType, ok := scalarConversionMap[dataValue.Kind()]; ok {
		newData = dataValue.Convert(convertType).Interface()
	} else {
		// chan, func, or something else. just pass it through
		newData = dataValue.Interface()
	}

	return dataValue.Interface(), newData, map[string]interface{}{}
}
