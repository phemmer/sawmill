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

type Event struct {
	Level Level
	Timestamp time.Time
	Message string
	Fields interface{}
}
