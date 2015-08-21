package event

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"reflect"
	"testing"
	"time"
)

type test struct {
	input        interface{}
	outputCopy   interface{}
	outputScalar interface{}
	outputFields map[string]interface{}
}

type simpleBool bool

type boolStringer bool

func (bs boolStringer) String() string {
	return fmt.Sprintf("%t!", bs)
}

type boolPointerStringer bool

func (bs *boolPointerStringer) String() string {
	return fmt.Sprintf("%t!", *bs)
}

type int64Errorer int64

func (be int64Errorer) Error() string {
	return "ERROR"
}

type int64PointerErrorer int64

func (be *int64PointerErrorer) Error() string {
	return "ERROR"
}

var nilPointer *bool

var tests = []test{
	// {
	//   input,
	//   copy,
	//   scalar,
	//   fields,
	// },
	{
		nil,
		nil,
		nil,
		map[string]interface{}{},
	},
	{
		7457,
		7457,
		7457,
		map[string]interface{}{},
	},
	{
		int64(7457),
		int64(7457),
		int64(7457),
		map[string]interface{}{},
	},
	{
		int8(127),
		int8(127),
		int8(127),
		map[string]interface{}{},
	},
	{
		uint8(255),
		uint8(255),
		uint8(255),
		map[string]interface{}{},
	},
	{
		"foo",
		"foo",
		"foo",
		map[string]interface{}{},
	},
	{
		nilPointer,
		nilPointer,
		nil,
		map[string]interface{}{},
	},
	{
		map[string]interface{}{"foo": (*time.Time)(nil)}, // wrap a nil pointer in an interface. the map is just the simplest way to do so
		map[interface{}]interface{}{"foo": (*time.Time)(nil)},
		nil,
		map[string]interface{}{},
	},
	{
		make(chan bool),
		nil,
		nil,
		map[string]interface{}{},
	},
	{
		func() bool { return false },
		nil,
		nil,
		map[string]interface{}{},
	},
	{
		func() *int { v := 7457; return &v }(),
		func() *int { v := 7457; return &v }(),
		7457,
		map[string]interface{}{},
	},
	{
		simpleBool(true),
		simpleBool(true),
		true,
		map[string]interface{}{},
	},
	{
		simpleBool(false),
		simpleBool(false),
		false,
		map[string]interface{}{},
	},
	{
		(*boolStringer)(nil),
		(*boolStringer)(nil),
		nil,
		map[string]interface{}{},
	},
	{
		map[string]string{"foo": "bar"},
		map[interface{}]interface{}{"foo": "bar"},
		nil,
		map[string]interface{}{"foo": "bar"},
	},
	{
		map[string]string{"foo": "bar", "pop": "tart"},
		map[interface{}]interface{}{"foo": "bar", "pop": "tart"},
		nil,
		map[string]interface{}{"foo": "bar", "pop": "tart"},
	},
	{
		map[interface{}]interface{}{"foo": "bar", "pop": "tart"},
		map[interface{}]interface{}{"foo": "bar", "pop": "tart"},
		nil,
		map[string]interface{}{"foo": "bar", "pop": "tart"},
	},
	{
		struct{ Foo string }{Foo: "bar"},
		map[string]interface{}{"Foo": "bar"},
		nil,
		map[string]interface{}{"Foo": "bar"},
	},
	{
		struct {
			Foo string
			Pop int
		}{Foo: "bar", Pop: 7457},
		map[string]interface{}{"Foo": "bar", "Pop": 7457},
		nil,
		map[string]interface{}{"Foo": "bar", "Pop": 7457},
	},
	{
		struct {
			Foo string
			Pop uint16
		}{"bar", 7457},
		map[string]interface{}{"Foo": "bar", "Pop": uint16(7457)},
		nil,
		map[string]interface{}{"Foo": "bar", "Pop": uint16(7457)},
	},
	{
		boolStringer(true),
		boolStringer(true),
		"true!",
		map[string]interface{}{},
	},
	{
		func() *boolPointerStringer { ptr := boolPointerStringer(true); return &ptr }(),
		func() *boolPointerStringer { ptr := boolPointerStringer(true); return &ptr }(),
		"true!",
		map[string]interface{}{},
	},
	{
		int64Errorer(1234),
		int64Errorer(1234),
		"ERROR",
		map[string]interface{}{},
	},
	{
		func() *int64Errorer { v := int64Errorer(1234); return &v }(),
		func() *int64Errorer { v := int64Errorer(1234); return &v }(),
		"ERROR",
		map[string]interface{}{},
	},
	{
		errors.New("FOO"),
		&map[string]interface{}{},
		"FOO",
		map[string]interface{}{},
	},
	{
		fmt.Errorf("FOO"),
		&map[string]interface{}{},
		"FOO",
		map[string]interface{}{},
	},
	{
		net.IPv4(192, 168, 0, 32),
		//net.IPv4(192, 168, 0, 32), //TODO make it so this is what it actually does do
		[]interface{}{byte(0), byte(0), byte(0), byte(0), byte(0), byte(0), byte(0), byte(0), byte(0), byte(0), byte(255), byte(255), byte(192), byte(168), byte(0), byte(32)},
		"192.168.0.32",
		map[string]interface{}{},
	},
	{
		map[string]interface{}{"foo": map[string]interface{}{"bar": "baz", "pop": "tart"}, "pop": "sicle"},
		//TODO make this return map[string]interface{}
		map[interface{}]interface{}{"foo": map[interface{}]interface{}{"bar": "baz", "pop": "tart"}, "pop": "sicle"},
		nil,
		map[string]interface{}{"foo.bar": "baz", "foo.pop": "tart", "pop": "sicle"},
	},
	{
		map[string]interface{}{"foo": map[string]int64Errorer{"bar": int64Errorer(1234), "baz": int64Errorer(0)}, "pop": "tart"},
		map[interface{}]interface{}{"foo": map[interface{}]interface{}{"bar": int64Errorer(1234), "baz": int64Errorer(0)}, "pop": "tart"},
		nil,
		map[string]interface{}{"foo.bar": "ERROR", "foo.baz": "ERROR", "pop": "tart"},
	},
}

func TestDeStruct(t *testing.T) {
	for i, test := range tests {
		outputCopy, outputScalar, outputFields := deStruct(test.input)

		testInputPtr := reflect.ValueOf(&test.input).Pointer()
		outputCopyPtr := reflect.ValueOf(&outputCopy).Pointer()

		assertionMessage := fmt.Sprintf("Test #%d: %#v", i, test.input)

		assert.NotEqual(t, testInputPtr, outputCopyPtr, assertionMessage)
		// tests commented pending fix for https://github.com/stretchr/testify/issues/144
		//assert.IsType(t, test.outputCopy, outputCopy, assertionMessage)
		//assert.Exactly(t, test.outputCopy, outputCopy, assertionMessage)
		if !reflect.DeepEqual(test.outputCopy, outputCopy) {
			assert.Fail(t, fmt.Sprintf("Not equal: %#v (expected)\n"+
				"        != %#v (actual)", test.outputCopy, outputCopy), assertionMessage)
		}
		assert.Equal(t, test.outputScalar, outputScalar, assertionMessage)
		//assert.Equal(t, test.outputFields, outputFields, assertionMessage)
		if !reflect.DeepEqual(test.outputFields, outputFields) {
			assert.Fail(t, fmt.Sprintf("Not equal: %#v (expected)\n"+
				"        != %#v (actual)", test.outputFields, outputFields), assertionMessage)
		}
	}
}

func BenchmarkDeStruct(b *testing.B) {
	var outputCopy interface{}
	var outputScalar interface{}
	var outputFields map[string]interface{}

	for i := 0; i < b.N; i++ {
		for _, test := range tests {
			outputCopy, outputScalar, outputFields = deStruct(test.input)
		}
	}

	if false { // keep the compiler from complaining about unused variables
		fmt.Printf("%v %v %v", outputCopy, outputScalar, outputFields)
	}
}
