package event

import (
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

var levelNames = [8]string{
	"Debug",
	"Info",
	"Notice",
	"Warning",
	"Error",
	"Critical",
	"Alert",
	"Emergency",
}

// LevelName converts a level into its textual representation.
func LevelName(level Level) string {
	return levelNames[level]
}

type Event struct {
	Id         uint64
	Level      Level
	Time       time.Time
	Message    string
	Fields     interface{}
	FlatFields map[string]interface{}
}

// NewEvent creates a new Event object.
// The time is set to current time, and the fields are deep-copied.
func NewEvent(id uint64, level Level, message string, fields interface{}) *Event {
	now := time.Now()

	fieldsCopy, _, flatFields := deStruct(fields)

	event := &Event{
		Id:         id,
		Time:       now,
		Level:      level,
		Message:    message,
		Fields:     fieldsCopy,
		FlatFields: flatFields,
	}

	return event
}

// LevelName returns the textual representation of the level name for the event.
func (event *Event) LevelName() string {
	return LevelName(event.Level)
}
