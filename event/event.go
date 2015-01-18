package event

import (
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
	Level Level
	Timestamp time.Time
	Message string
	Fields interface{}
}

func (event *Event) LevelName() string {
	return LevelName(event.Level)
}
