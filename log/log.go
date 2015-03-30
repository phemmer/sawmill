/*
The log package provides stdlib log interface compatability to sawmill.

Its purpose is to ease transition into sawmill, not be used as the main interface.

Any events logged with this package are sent to the sawmill package-level logger (e.g. `sawmill.Info()`)
*/
package log

import (
	"fmt"
	"io"
	"os"
	"sync"

	sm "github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/handler/writer"
)

const ( // currently unused. here for compatability
	Ldate         = 1 << iota     // the date: 2009/01/23
	Ltime                         // the time: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

var stdPrefix string
var stdPrefixMutex sync.Mutex

func Fatal(v ...interface{}) {
	sm.Sync(sm.Event(sm.CriticalLevel, Prefix()+fmt.Sprint(v...)))
	sm.Stop()
	os.Exit(1)
}
func Fatalf(format string, v ...interface{}) {
	sm.Sync(sm.Event(sm.CriticalLevel, Prefix()+fmt.Sprintf(format, v...)))
	sm.Stop()
	os.Exit(1)
}
func Fatalln(v ...interface{}) {
	sm.Sync(sm.Event(sm.CriticalLevel, Prefix()+fmt.Sprintln(v...)))
	sm.Stop()
	os.Exit(1)
}
func Flags() int {
	return 0
}
func Panic(v ...interface{}) {
	message := fmt.Sprint(v...)
	sm.Sync(sm.Event(sm.CriticalLevel, Prefix()+message))
	sm.Stop()
	panic(message)
}
func Panicf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	sm.Sync(sm.Event(sm.CriticalLevel, Prefix()+message))
	sm.Stop()
	panic(message)
}
func Panicln(v ...interface{}) {
	message := fmt.Sprintln(v...)
	sm.Sync(sm.Event(sm.CriticalLevel, Prefix()+message))
	sm.Stop()
	panic(message)
}
func Prefix() string {
	stdPrefixMutex.Lock()
	prefix := stdPrefix
	stdPrefixMutex.Unlock()
	return prefix
}
func Print(v ...interface{}) {
	sm.Sync(sm.Event(sm.InfoLevel, Prefix()+fmt.Sprint(v...)))
}
func Printf(format string, v ...interface{}) {
	sm.Sync(sm.Event(sm.InfoLevel, Prefix()+fmt.Sprintf(format, v...)))
}
func Println(v ...interface{}) {
	sm.Sync(sm.Event(sm.InfoLevel, Prefix()+fmt.Sprintln(v...)))
}
func SetFlags(flag int) {
	//TODO
}
func SetOutput(w io.Writer) {
	//TODO?
}
func SetPrefix(prefix string) {
	stdPrefixMutex.Lock()
	stdPrefix = prefix
	stdPrefixMutex.Unlock()
}

type Logger struct {
	sml         *sm.Logger
	prefix      string
	prefixMutex sync.Mutex
}

func New(out io.Writer, prefix string, flag int) *Logger {
	sml := sm.NewLogger()

	handler, _ := writer.New(out, event.ConsoleNocolorFormat)

	sml.AddHandler("logwriter", handler)

	return &Logger{sml: sml, prefix: prefix}
}
func (l *Logger) Fatal(v ...interface{}) {
	l.sml.Sync(l.sml.Event(sm.CriticalLevel, fmt.Sprint(v...)))
	l.sml.Stop()
	os.Exit(1)
}
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.sml.Sync(l.sml.Event(sm.CriticalLevel, fmt.Sprintf(format, v...)))
	l.sml.Stop()
	os.Exit(1)
}
func (l *Logger) Fatalln(v ...interface{}) {
	l.sml.Sync(l.sml.Event(sm.CriticalLevel, fmt.Sprintln(v...)))
	l.sml.Stop()
	os.Exit(1)
}
func (l *Logger) Flags() int {
	return 0
}
func (l *Logger) Panic(v ...interface{}) {
	message := fmt.Sprint(v...)
	l.sml.Sync(l.sml.Event(sm.CriticalLevel, message))
	l.sml.Stop()
	panic(message)
}
func (l *Logger) Panicf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	l.sml.Sync(l.sml.Event(sm.CriticalLevel, message))
	l.sml.Stop()
	panic(message)
}
func (l *Logger) Panicln(v ...interface{}) {
	message := fmt.Sprintln(v...)
	l.sml.Sync(l.sml.Event(sm.CriticalLevel, message))
	l.sml.Stop()
	panic(message)
}
func (l *Logger) Prefix() string {
	l.prefixMutex.Lock()
	prefix := l.prefix
	l.prefixMutex.Unlock()
	return prefix
}
func (l *Logger) Print(v ...interface{}) {
	l.sml.Sync(l.sml.Event(sm.InfoLevel, l.Prefix()+fmt.Sprint(v...)))
}
func (l *Logger) Printf(format string, v ...interface{}) {
	l.sml.Sync(l.sml.Event(sm.InfoLevel, l.Prefix()+fmt.Sprintf(format, v...)))
}
func (l *Logger) Println(v ...interface{}) {
	l.sml.Sync(l.sml.Event(sm.InfoLevel, l.Prefix()+fmt.Sprintln(v...)))
}
func (l *Logger) SetFlags(flag int) {
	//TODO
}
func (l *Logger) SetOutput(w io.Writer) {
	//TODO
}
func (l *Logger) SetPrefix(prefix string) {
	l.prefixMutex.Lock()
	l.prefix = prefix
	l.prefixMutex.Unlock()
}
