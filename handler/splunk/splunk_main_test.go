package splunk

import (
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type splunkServerToken struct {
	SessionKey string
	expires    time.Time
}
type splunkServerEvent struct {
	message    string
	host       string
	source     string
	sourcetype string
	index      string
}
type splunkServer struct {
	tokens []*splunkServerToken
	events []*splunkServerEvent
}

func (server *splunkServer) makeToken() *splunkServerToken {
	token := &splunkServerToken{
		SessionKey: strconv.FormatInt(rand.Int63(), 36),
		expires:    time.Now().Add(time.Minute * 60),
	}
	server.tokens = append(server.tokens, token)
	return token
}
func (server *splunkServer) checkToken(sessionKey string) bool {
	for _, token := range server.tokens {
		if token.SessionKey == sessionKey && token.expires.After(time.Now()) {
			return true
		}
	}
	return false
}

func (server *splunkServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/services/auth/login" {
		server.ServeHTTPLogin(w, r)
	} else if r.URL.Path == "/services/receivers/simple" {
		server.ServeHTTPReceiversSimple(w, r)
	}
	return
}
func (server *splunkServer) ServeHTTPLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if !(r.Form.Get("username") == "admin" && r.Form.Get("password") == "knockknock") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(server.makeToken())
}
func (server *splunkServer) ServeHTTPReceiversSimple(w http.ResponseWriter, r *http.Request) {
	authorization := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(authorization) < 2 || authorization[0] != "Splunk" || !server.checkToken(authorization[1]) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	event := &splunkServerEvent{}
	body, _ := ioutil.ReadAll(r.Body)
	event.message = string(body)
	event.index = r.URL.Query().Get("index")
	event.host = r.URL.Query().Get("host")
	event.source = r.URL.Query().Get("source")
	event.sourcetype = r.URL.Query().Get("sourcetype")
	server.events = append(server.events, event)
}

var splunkSvr = &splunkServer{}
var splunkHttpSvr *httptest.Server
var splunkHttpsSvr *httptest.Server

func splunkURL(svr *httptest.Server) string {
	svrURL, _ := url.Parse(svr.URL)
	svrURL.User = url.UserPassword("admin", "knockknock")
	return svrURL.String()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}
func testMain(m *testing.M) int {
	splunkHttpsSvr = httptest.NewTLSServer(splunkSvr)
	defer splunkHttpsSvr.Close()
	cert, _ := x509.ParseCertificate(splunkHttpsSvr.TLS.Certificates[0].Certificate[0])
	CACerts.AddCert(cert)
	// getHttpsClient causes the server to dump a TLS error during the probe. so make it shut up
	splunkHttpsSvr.Config.ErrorLog = log.New(ioutil.Discard, "", 0)

	splunkHttpSvr = httptest.NewServer(splunkSvr)
	defer splunkHttpSvr.Close()

	return m.Run()
}
