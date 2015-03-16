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

	dataCopy, _, flatFields := deStruct(data)

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
