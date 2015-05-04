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
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/util"
)

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

	repoRoot, repoRevision, repoTag := util.RepoInfo()
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
		filePath := frame.File
		inApp := false
		if repoPath != "" && strings.HasPrefix(filePath, repoPath) {
			filePath = filePath[len(repoPath):]

			if !strings.Contains(frame.File, "/Godeps/_workspace/src/") {
				//TODO come up with a better way of filtering external packages
				inApp = true
			}
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

// Event sends the given log event to the sentry service.
func (s *Sentry) Event(logEvent *event.Event) error {
	message := logEvent.Message
	if err, ok := logEvent.FlatFields["error"]; ok {
		message = fmt.Sprintf("%s: %v", message, err)
	} else if err, ok := logEvent.FlatFields["err"]; ok {
		message = fmt.Sprintf("%s: %v", message, err)
	}

	s.RLock()
	packet := raven.NewPacket(message)
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
