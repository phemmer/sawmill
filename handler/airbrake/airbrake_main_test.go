package airbrake

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/phemmer/sawmill/event"
)

func init() {
	event.RepoPath = event.FilePath
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

var testAirbrakeServer *airbrakeSvr

func testMain(m *testing.M) int {
	testAirbrakeServer = &airbrakeSvr{
		projectID:  "123456",
		projectKey: "0123456789abcdef0123456789abcdef",
	}
	httpServer := httptest.NewServer(testAirbrakeServer)
	defer httpServer.Close()
	testAirbrakeServer.httpServer = httpServer
	airbrakeURL = httpServer.URL
	return m.Run()
}

type airbrakeSvr struct {
	projectID  string
	projectKey string

	httpServer *httptest.Server

	notices []map[string]interface{}
}

func (server *airbrakeSvr) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	projectID := pathParts[4]
	projectKey := r.URL.Query().Get("key")

	if server.projectID != projectID || server.projectKey != projectKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	notice := map[string]interface{}{}
	err := json.NewDecoder(r.Body).Decode(&notice)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	server.notices = append(server.notices, notice)

	locateURL, _ := url.Parse(r.URL.String())
	locateURL.Path = fmt.Sprintf("/locate/%s", id)
	locateURL.RawQuery = ""
	resp := map[string]string{
		"id":  id,
		"url": locateURL.String(),
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
