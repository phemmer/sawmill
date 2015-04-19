package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEvent(t *testing.T) {
	e := NewEvent(
		123,
		Notice,
		"test NewEvent",
		map[string]interface{}{"foo": map[string]string{"bar": "baz"}},
	)

	assert.Equal(t, 123, e.Id)
	assert.Equal(t, Notice, e.Level)
	assert.Equal(t, "test NewEvent", e.Message)
	assert.Equal(t, map[string]interface{}{"foo.bar": "baz"}, e.FlatFields)
	assert.Equal(t, map[interface{}]interface{}{"foo": map[interface{}]interface{}{"bar": "baz"}}, e.Fields)
	assert.WithinDuration(t, time.Now(), e.Time, time.Second)
}
