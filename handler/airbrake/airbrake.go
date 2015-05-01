// The airbrake package provides a handler which sends events to the Airbrake
// error reporting service.
//
// For a stack trace to be included, sawmill needs to be configured to gather
// them. E.G.:
//  logger.SetStackMinLevel(sawmill.ErrorLevel)
//
// The handler sends all received events to the airbrake service. Thus it
// should most likely be used in combination with the filter handler
// (http://godoc.org/github.com/phemmer/sawmill/handler/filter).
package airbrake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sync"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/util"
)

var airbrakeURL = "https://airbrake.io"

var notifier = airbrakeNotifier{
	Name:    "sawmill",
	Version: "0.1",
	Url:     "https://github.com/phemmer/sawmill",
}

type airbrakeNotifier struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Url     string `json:"url"`
}

type airbrakeError struct {
	ErrType   string                `json:"type"`
	Message   string                `json:"message"`
	Backtrace []*airbrakeStackFrame `json:"backtrace"`
}

type AirbrakeContext struct {
	OS          string `json:"os"`
	Language    string `json:"language"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	URL         string `json:"url"`

	// RootDirectory should be the project directory. This is normally
	// automatically deteremined.
	// If airbrake is integrated with your version control system (github), it
	// will generate links for you by stripping the RootDirectory from the
	// backtrace path, and adding it to the VCS repo URL.
	RootDirectory string `json:"rootDirectory"`
	UserId        string `json:"userId"`
	UserName      string `json:"userName"`
	UserEmail     string `json:"userEmail"`
}

type airbrakeNotice struct {
	Notifier    *airbrakeNotifier      `json:"notifier"`
	Errors      []*airbrakeError       `json:"errors"`
	Context     *AirbrakeContext       `json:"context"`
	Environment map[string]string      `json:"environment"`
	Session     map[string]interface{} `json:"session"`
	Params      map[string]interface{} `json:"params"`
}

type airbrakeStackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

func newAirbrakeStack(stack []*event.StackFrame) []*airbrakeStackFrame {
	asf := make([]*airbrakeStackFrame, len(stack))

	for i, frame := range stack {
		asf[i] = &airbrakeStackFrame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Func,
		}
	}

	return asf
}

// AirbrakeHandler implements the sawmill.Handler interface.
//
// Modifying attributes after adding to a logger should be performed only after
// calling AirbrakeHandler.Lock() (and Unlock() after complete).
// However before being added to a sawmill logger, attributes may be modified
// without locking.
type AirbrakeHandler struct {
	sync.Mutex

	// AirbrakeURL is the base URL for the airbrake service
	// (e.g. https://airbrake.io).
	AirbrakeURL *url.URL
	projectId   int64
	key         string

	// Context contains the fields which show up under the "context" section of
	// airbrake.
	// Some fields are automatically populated:
	//  * OS - Obtained from GOOS & GOARCH.
	//  * Language - "go" & the go compiler version.
	//  * Environment - The 'environment' parameter passed to New().
	//  * RootDirectory - The VCS repo root of the caller of New().
	//  * Version - The tag or commit of the caller's repo (git only).
	Context *AirbrakeContext

	// Env is an arbitrary mapping of environmental variables.
	// For security reasons, the OS environment variables are not copied.
	// Instead a few common variables are provided instead.
	//  * _ - The name of the process
	//  * GOVERSION - The go compiler version
	//  * GOMAXPROCS - The value of GOMAXPROCS at the time of New()
	//  * GOROOT - The value of GOROOT
	//  * HOSTNAME - The system's host name
	// Values may be added or removed.
	Env map[string]string
}

// New constructs a new AirbrakeHandler.
//
// The projectId & key parameters should be obtained from airbrake. The
// environment parameter is the name of the environment to report errors under.
func New(projectId int64, key string, environment string) *AirbrakeHandler {
	repoRoot, _, repoVersion := util.RepoInfo()
	context := &AirbrakeContext{
		OS:            runtime.GOOS + " " + runtime.GOARCH, //TODO syscall.Uname() when in linux
		Language:      "go " + runtime.Version(),
		Environment:   environment,
		RootDirectory: repoRoot,
		Version:       repoVersion,
	}

	env := map[string]string{}
	env["_"] = os.Args[0]
	env["GOVERSION"] = runtime.Version()
	env["GOMAXPROCS"] = fmt.Sprintf("%d", runtime.GOMAXPROCS(0))
	env["GOROOT"] = runtime.GOROOT()
	env["HOSTNAME"], _ = os.Hostname()

	aURL, _ := url.Parse(fmt.Sprintf("%s/api/v3/projects/%d/notices?key=%s", airbrakeURL, projectId, key))
	return &AirbrakeHandler{
		AirbrakeURL: aURL,
		projectId:   projectId,
		key:         key,
		Context:     context,
		Env:         env,
	}
}

// Event processes a sawmill event, and sends it to the airbrake service.
func (ah *AirbrakeHandler) Event(logEvent *event.Event) error {
	errors := []*airbrakeError{}

	aErr := &airbrakeError{
		//TODO once sawmill/event copies the original struct, we can access the type and expose it here
		ErrType:   logEvent.Level.String(),
		Message:   logEvent.Message,
		Backtrace: newAirbrakeStack(logEvent.Stack),
	}
	errors = append(errors, aErr)

	ah.Lock()
	notice := &airbrakeNotice{
		Notifier:    &notifier,
		Errors:      errors,
		Params:      logEvent.FlatFields,
		Context:     ah.Context,
		Environment: ah.Env,
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(notice); err != nil {
		ah.Unlock()
		return err
	}

	resp, err := http.Post(ah.AirbrakeURL.String(), "application/json", buf)
	ah.Unlock()
	if err != nil {
		return err
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code from airbrake: %d", resp.StatusCode)
	}

	return nil
}
