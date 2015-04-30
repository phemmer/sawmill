// The sentry package provides a handler which sends events to the Sentry
// exception reporting service.
//
// For a stack trace to be included, sawmill needs to be configured to gather
// them. E.G.:
//  logger.SetStackMinLevel(sawmill.ErrorLevel)
//
// The handler sends all received events to the sentry service. Thus it
// should most likely be used in combination with the filter handler
// (http://godoc.org/github.com/phemmer/sawmill/handler/filter).
package sentry

import (
	"bytes"
	"math/rand"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/phemmer/sawmill/event"
	"golang.org/x/tools/go/vcs"
)

// filePath is the path to this file.
// This is used when getting the VCS repo information from the caller.
var filePath string

func init() {
	_, filePath, _, _ = runtime.Caller(0)
}

type Sentry struct {
	sync.RWMutex
	idPrefix     string
	repoRoot     string
	repoRevision string
	repoTag      string
	client       *raven.Client
}

// New constructs a new Sentry handler.
//
// The DSN is provided by the Sentry service.
func New(dsn string) (*Sentry, error) {
	client, err := raven.NewClient(dsn, map[string]string{})
	if err != nil {
		return nil, err
	}

	repoRoot, repoRevision, repoTag := repoInfo()
	client.SetRelease(repoTag)
	if repoRevision != "" {
		client.Tags["revision"] = repoRevision
	}

	// generate a random value to use as the ID prefix.
	// We want to use a common prefix for all events from this handler.
	prefix := rand.New(rand.NewSource(time.Now().UnixNano())).Int63()

	return &Sentry{
		idPrefix:     strconv.FormatInt(prefix, 36) + ".",
		repoRoot:     repoRoot,
		repoRevision: repoRevision,
		repoTag:      repoTag,
		client:       client,
	}, nil
}

// Tag adds a tag which will be applied to all sentry packets.
func (s *Sentry) Tag(key, value string) {
	s.Lock()
	s.client.Tags[key] = value
	s.Unlock()
}

// Untag removes a stored tag.
func (s *Sentry) Untag(key string) {
	s.Lock()
	delete(s.client.Tags, key)
	s.Unlock()
}

// ravenLevels is a map of sawmill event levels to raven levels.
var ravenLevels = map[event.Level]raven.Severity{
	event.Debug:     raven.DEBUG,
	event.Info:      raven.INFO,
	event.Notice:    raven.INFO,
	event.Warning:   raven.WARNING,
	event.Error:     raven.ERROR,
	event.Critical:  raven.FATAL,
	event.Alert:     raven.FATAL,
	event.Emergency: raven.FATAL,
}

// ravenTrace converts a sawmill stack trace into a raven stack trace.
func ravenTrace(repoPath string, stack []*event.StackFrame) *raven.Stacktrace {
	if repoPath != "" && repoPath[len(repoPath)-1] != '/' {
		repoPath = repoPath + "/"
	}
	stackLen := len(stack)
	ravenFrames := make([]*raven.StacktraceFrame, stackLen)
	for i, frame := range stack {
		filePath = frame.File
		inApp := false
		if repoPath != "" && strings.HasPrefix(filePath, repoPath) {
			filePath = filePath[len(repoPath):]
			inApp = true
		}

		sourceBefore, source, sourceAfter := frame.SourceContext(3, 0)
		sourceBeforeStrs := make([]string, len(sourceBefore))
		for i, line := range sourceBefore {
			sourceBeforeStrs[i] = string(line)
		}
		sourceAfterStrs := make([]string, len(sourceAfter))
		for i, line := range sourceAfter {
			sourceAfterStrs[i] = string(line)
		}

		ravenFrame := &raven.StacktraceFrame{
			Filename:     filePath,
			Function:     frame.Func,
			Module:       frame.Package,
			Lineno:       frame.Line,
			AbsolutePath: frame.File,
			InApp:        inApp,
			ContextLine:  string(source),
			PreContext:   sourceBeforeStrs,
			PostContext:  sourceAfterStrs,
		}
		ravenFrames[stackLen-1-i] = ravenFrame
	}

	return &raven.Stacktrace{Frames: ravenFrames}
}

// repoInfo attempts to find the repo information for the caller, and returns
// the path to the top of the repo, the commit ID, and the tag.
func repoInfo() (repoRoot string, repoRevision string, repoTag string) {
	var vcsCmd *vcs.Cmd

	callers := make([]uintptr, 10)
	callers = callers[:runtime.Callers(2, callers)]
	for _, caller := range callers {
		f := runtime.FuncForPC(caller)
		file, _ := f.FileLine(caller)
		if file == filePath {
			continue
		}

		// walk up the dir until we get to the first directory in root.
		// This is an unfortunately necessity to use vcs.FromDir()
		topDir := file
		for dir := path.Dir(topDir); dir != "." && dir[len(dir)-1] != '/' && dir != topDir; dir = path.Dir(topDir) {
			topDir = dir
		}

		var err error
		vcsCmd, repoRoot, err = vcs.FromDir(path.Dir(file), topDir)
		if err != nil {
			repoRoot = path.Dir(file)
		}
		repoRoot = topDir + "/" + repoRoot
		break
	}

	if vcsCmd != nil && vcsCmd.Name == "Git" {
		execCmd := exec.Command("git", "describe", "--dirty", "--match", "", "--always")
		execCmd.Dir = repoRoot
		output, err := execCmd.Output()
		if err == nil {
			repoRevision = string(bytes.TrimRight(output, "\n"))
		}

		execCmd = exec.Command("git", "describe", "--dirty", "--tags", "--always")
		execCmd.Dir = repoRoot
		output, err = execCmd.Output()
		if err == nil {
			repoTag = string(bytes.TrimRight(output, "\n"))
		}
	}

	return
}

// Event sends the given log event to the sentry service.
func (s *Sentry) Event(logEvent *event.Event) error {
	s.RLock()
	packet := raven.NewPacket(logEvent.Message)
	packet.Logger = "sawmill"
	packet.EventID = s.idPrefix + strconv.FormatInt(int64(logEvent.Id), 10)
	packet.Timestamp = raven.Timestamp(logEvent.Time)
	packet.Level = ravenLevels[logEvent.Level]
	for k, v := range logEvent.FlatFields {
		packet.Extra[k] = v
	}

	if len(logEvent.Stack) != 0 {
		packet.Culprit = logEvent.Stack[0].Package + "." + logEvent.Stack[0].Func
	}

	// translate logEvent.Stack into raven.Stacktrace
	packet.Interfaces = append(packet.Interfaces, ravenTrace(s.repoRoot, logEvent.Stack))

	_, errChan := s.client.Capture(packet, nil)
	err := <-errChan
	s.RUnlock()
	return err
}

func (s *Sentry) Stop() {
	s.client.Close()
}
