package event

/*
import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

type strStruct struct{}

func (s strStruct) String() string {
	return "STRING"
}

func Test_deStruct_string_interface(t *testing.T) {
	str := &strStruct{}

	fields := map[string]interface{}{}
	strCopy := deStruct(str, "parent", fields)

	assert.Equal(t, "STRING", strCopy)
	assert.Equal(t, "STRING", fields["parent"])
}

////////////////////////////////////////

type errStruct struct{}

func (e errStruct) Error() string {
	return "ERROR"
}

func Test_deStruct_error(t *testing.T) {
	err := &errStruct{}

	fields := map[string]interface{}{}
	errCopy := deStruct(err, "parent", fields)

	assert.Equal(t, "ERROR", errCopy)
	assert.Equal(t, "ERROR", fields["parent"])
}

////////////////////////////////////////

type errStructWithExports struct {
	Attribute bool
}

func (e errStructWithExports) Error() string {
	return "ERROR"
}

// Check that if we have a struct with exported attributes, and that satisfies the error interface, that we expose the attributes, and then the error message as an `Error` attribute
func Test_deStruct_errorWithExports(t *testing.T) {
	err := &errStructWithExports{true}

	fields := map[string]interface{}{}
	errCopy := deStruct(err, "parent", fields)

	errCopyMap, ok := errCopy.(map[string]interface{})
	if assert.True(t, ok) {
		assert.Equal(t, "ERROR", errCopyMap["Error"])
	}

	assert.Equal(t, "ERROR", fields["parent.Error"])
}

////////////////////////////////////////

// Check that if we have a struct where the Error() is on the pointer, that we still work

func Test_deStruct_errorNonPointer(t *testing.T) {
	err := errors.New("ERROR")

	fields := map[string]interface{}{}
	errCopy := deStruct(err, "parent", fields)

	errCopyMap, ok := errCopy.(map[string]interface{})
	if assert.True(t, ok) {
		assert.Equal(t, "ERROR", errCopyMap["Error"])
	}

	assert.Equal(t, "ERROR", fields["parent.Error"])
}
*/
