package event

import (
	"bufio"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

var stackMaxDepth = 100

// FilePath is the full path to this source file.
//
// This is meant to be used for internal sawmill testing only.
var FilePath string // used for testing

// RepoPath is the path to the top of the sawmill repo.
//
// This is meant to be used for internal sawmill testing only.
//
// The value is used when obtaining a stack trace to determine where the trace
// should start. The first frame not in RepoPath is considered to be the top.
var RepoPath string

func init() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return
	}

	FilePath = file
	RepoPath = path.Dir(file)
	// the next trimRight fixes an issue when using `go test -cover`
	// https://groups.google.com/forum/#!topic/golang-nuts/eL6n8au6PAw
	RepoPath = strings.TrimSuffix(RepoPath, "/_test/_obj_test")
	RepoPath = path.Dir(RepoPath)
}

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
	"debug",
	"info",
	"notice",
	"warning",
	"error",
	"critical",
	"alert",
	"emergency",
}

func (l Level) String() string {
	return levelNames[l]
}
func (l Level) Int() int {
	return int(l)
}

type Event struct {
	Id         uint64
	Level      Level
	Time       time.Time
	Message    string
	Fields     interface{}
	FlatFields map[string]interface{}
	Stack      []*StackFrame
}

// StackFrame describes an entry in a call stack.
type StackFrame struct {
	PC       uintptr
	File     string
	Line     int
	Function string
	Func     string
	Package  string
}

func newStackFrame(pc uintptr) *StackFrame {
	f := runtime.FuncForPC(pc)
	if f == nil {
		return nil
	}

	// get the PC of the instruction, not the return
	// https://github.com/golang/go/issues/5518
	// https://play.golang.org/p/lTWpWsrIT3
	linePC := pc
	if linePC > f.Entry() {
		linePC--
	}

	file, line := f.FileLine(linePC)
	fSplit := strings.SplitN(path.Base(f.Name()), ".", 2)
	pkg, fun := fSplit[0], fSplit[1]
	return &StackFrame{
		PC:       pc,
		File:     file,
		Line:     line,
		Function: f.Name(),
		Package:  pkg,
		Func:     fun,
	}
}

// Source returns the source code line of the stack frame.
// If the source cannot be read for any reason, nil is returned.
func (sf *StackFrame) Source() []byte {
	file, err := os.Open(sf.File)
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(file)
	for i := 0; i < sf.Line; i++ {
		if !scanner.Scan() {
			file.Close()
			return nil
		}
	}
	file.Close()
	return scanner.Bytes()
}

// SourceContext retuns the source code lines surrounding the stack frame.
func (sf *StackFrame) SourceContext(beforeCount int, afterCount int) (linesBefore [][]byte, line []byte, linesAfter [][]byte) {

	if beforeCount >= sf.Line {
		beforeCount = sf.Line - 1
	}
	firstLine := sf.Line - beforeCount
	lastLine := sf.Line + afterCount

	lines := [][]byte{}

	file, err := os.Open(sf.File)
	if err != nil {
		return nil, nil, nil
	}

	scanner := bufio.NewScanner(file)
	for i := 1; scanner.Scan(); i++ {
		if i > lastLine {
			break
		}
		if i < firstLine {
			continue
		}
		lines = append(lines, scanner.Bytes())
	}
	file.Close()

	if len(lines) < beforeCount+1 {
		// something went wrong. We didn't read the whole source
		return nil, nil, nil
	}

	linesBefore = lines[:beforeCount]
	line = lines[beforeCount]
	linesAfter = lines[beforeCount+1:]

	return linesBefore, line, linesAfter
}

// New creates a new Event object.
// The time is set to current time, and the fields are deep-copied.
func New(id uint64, level Level, message string, fields interface{}, getStack bool) *Event {
	now := time.Now()

	var stack []*StackFrame
	if getStack {
		callers := make([]uintptr, stackMaxDepth)
		n := runtime.Callers(1, callers)
		callers = callers[:n]
		for i, caller := range callers {
			f := runtime.FuncForPC(caller)
			if file, _ := f.FileLine(caller); strings.HasPrefix(file, RepoPath) {
				continue
			}
			callers = callers[i:]
			break
		}
		stack = make([]*StackFrame, len(callers))
		for i, caller := range callers {
			stack[i] = newStackFrame(caller)
		}
	}

	fieldsCopy, _, flatFields := deStruct(fields)

	event := &Event{
		Id:         id,
		Time:       now,
		Level:      level,
		Message:    message,
		Fields:     fieldsCopy,
		FlatFields: flatFields,
		Stack:      stack,
	}

	return event
}
