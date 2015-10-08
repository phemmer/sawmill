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

func (ss *splunkServer) makeToken() *splunkServerToken {
	t := &splunkServerToken{
		SessionKey: strconv.FormatInt(rand.Int63(), 36),
		expires:    time.Now().Add(time.Minute * 60),
	}
	ss.tokens = append(ss.tokens, t)
	return t
}

func (ss *splunkServer) checkToken(sessionKey string) bool {
	for _, t := range ss.tokens {
		if t.SessionKey == sessionKey && t.expires.After(time.Now()) {
			return true
		}
	}
	return false
}

func (ss *splunkServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/services/auth/login" {
		ss.ServeHTTPLogin(w, r)
	} else if r.URL.Path == "/services/receivers/simple" {
		ss.ServeHTTPReceiversSimple(w, r)
	}
	return
}

func (ss *splunkServer) ServeHTTPLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if !(r.Form.Get("username") == "admin" && r.Form.Get("password") == "knockknock") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(ss.makeToken())
}

func (ss *splunkServer) ServeHTTPReceiversSimple(w http.ResponseWriter, r *http.Request) {
	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(auth) < 2 || auth[0] != "Splunk" || !ss.checkToken(auth[1]) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	e := &splunkServerEvent{
		message:    string(body),
		index:      r.URL.Query().Get("index"),
		host:       r.URL.Query().Get("host"),
		source:     r.URL.Query().Get("source"),
		sourcetype: r.URL.Query().Get("sourcetype"),
	}
	ss.events = append(ss.events, e)
}

var (
	splunkSvr      = &splunkServer{}
	splunkHttpSvr  *httptest.Server
	splunkHttpsSvr *httptest.Server
)

func splunkURL(s *httptest.Server) string {
	u, _ := url.Parse(s.URL)
	u.User = url.UserPassword("admin", "knockknock")
	return u.String()
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
