package airbrake

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemmer/sawmill/event"
)

func TestAirbrakeHandler(t *testing.T) {
	ah := New(123456, "0123456789abcdef0123456789abcdef", "testing")

	ah.Context.URL = "http://example.com/airbrake"
	ah.Context.UserId = "johndoe"
	ah.Context.UserName = "john doe"
	ah.Context.UserEmail = "john.doe@example.com"

	logEvent := event.NewEvent(1, event.Warning, "test AirbrakeHandler", map[string]interface{}{"foo": "bar"}, true)
	err := ah.Event(logEvent)
	require.NoError(t, err)

	notice := testAirbrakeServer.notices[len(testAirbrakeServer.notices)-1]

	nNotifier := notice["notifier"].(map[string]interface{})
	assert.Equal(t, notifier.Name, nNotifier["name"])
	assert.Equal(t, notifier.Version, nNotifier["version"])
	assert.Equal(t, notifier.Url, nNotifier["url"])

	_, selfFile, _, _ := runtime.Caller(0)
	nErrors := notice["errors"].([]interface{})
	nError := nErrors[0].(map[string]interface{})
	nBacktrace := nError["backtrace"].([]interface{})
	nBacktrace0 := nBacktrace[0].(map[string]interface{})
	assert.Equal(t, "Warning", nError["type"])
	assert.Equal(t, "test AirbrakeHandler", nError["message"])
	assert.Equal(t, selfFile, nBacktrace0["file"])
	assert.NotEmpty(t, nBacktrace0["line"])
	assert.Equal(t, "TestAirbrakeHandler", nBacktrace0["function"])

	var repoVersion string
	execCmd := exec.Command("git", "describe", "--dirty", "--tags", "--always")
	output, err := execCmd.Output()
	if err == nil {
		repoVersion = string(bytes.TrimRight(output, "\n"))
	}
	nContext := notice["context"].(map[string]interface{})
	assert.Equal(t, runtime.GOOS+" "+runtime.GOARCH, nContext["os"])
	assert.Equal(t, "go "+runtime.Version(), nContext["language"])
	assert.Equal(t, "testing", nContext["environment"])
	assert.Equal(t, repoVersion, nContext["version"])
	assert.Equal(t, "http://example.com/airbrake", nContext["url"])
	assert.Equal(t, "johndoe", nContext["userId"])
	assert.Equal(t, "john doe", nContext["userName"])
	assert.Equal(t, "john.doe@example.com", nContext["userEmail"])

	hostname, _ := os.Hostname()
	nEnvironment := notice["environment"].(map[string]interface{})
	assert.Equal(t, fmt.Sprintf("%d", runtime.GOMAXPROCS(0)), nEnvironment["GOMAXPROCS"])
	assert.Equal(t, runtime.Version(), nEnvironment["GOVERSION"])
	assert.Equal(t, runtime.GOROOT(), nEnvironment["GOROOT"])
	assert.Equal(t, hostname, nEnvironment["HOSTNAME"])

	nParams := notice["params"].(map[string]interface{})
	assert.Equal(t, "bar", nParams["foo"])
}

func TestRepoRoot(t *testing.T) {
	// assume this test file is `sawmill/handler/airbrake/airbrake_test.go`
	_, gitRoot, _, _ := runtime.Caller(0)
	gitRoot = path.Dir(path.Dir(path.Dir(gitRoot)))

	repoRoot, repoVersion := repoInfo()
	assert.Equal(t, gitRoot, repoRoot)
	assert.NotEmpty(t, repoVersion)
}
