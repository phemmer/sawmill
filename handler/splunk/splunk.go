/*
The splunk package is an event handler responsible for sending events to Splunk via the HTTP API.

In the event that the Splunk API endpoint uses HTTPS and a certificate not recognized by the standard certificate authorities, you may use add the server/CA cert to splunk.CACerts.
The CA cert for Splunk cloud is already recognized.

Template

The splunk template provides a few extra functions on top of the default sawmill event formatter template.

 Hostname - The system hostname (os.Hostname())
 Source - The application name (path.Base(os.Argv[0]))
 Pid - The process ID (os.Getpid())

*/
package splunk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
)

// SplunkFormat is the default template format.
// It is meant to work with the 'syslog' splunk sourcetype, such that the splunk field extraction matches most of the headers. The only header not properly parsed is the level.
const SplunkFormat = "{{.Time \"2006-01-02 15:04:05.000 -0700\"}} {{.Level}}({{.Event.Level}}) {{Source}}[{{Pid}}]: " + formatter.SIMPLE_FORMAT

// SplunkSourceType is the default splunk source type
const SplunkSourceType = "syslog"

// sessionKeyDuration is how long to use the same session key before requesting a new one.
const sessionKeyDuration = time.Duration(time.Minute * 15)

// All of the exported attribues are safe to replace before the handler has been added into a logger.
type SplunkHandler struct {
	url      *url.URL
	username string
	password string

	Template   *template.Template
	Index      string
	SourceType string
	Hostname   string
	Source     string

	client *http.Client

	sessionKey     string `json:"sessionKey"`
	sessionKeyTime time.Time
}

// New constructs a new splunk handler.
//
// The URL parameter is the URL of the Splunk API endpoint (e.g. https://user:pass@splunk.example.com:8089), and must contain authentication credentials.
// The URL may include a few query parameters which override default settings.
// * Index - The index to send events to. Default: "default"
// * SourceType - The source type to report log entries as. Default: "syslog"
// * Hostname - The hostname to report as the origin of the log entries. Default: os.Hostname()
// * Source - The source metadata parameter to send log entries with. Default: base(os.Argv[0])
//
// If the Splunk server uses https and has a cert not recognized by a standard certificate authority, you can use splunk.CACerts to add the CA/server certificate.
func New(splunkURL string) (*SplunkHandler, error) {
	sw := &SplunkHandler{}

	var err error
	sw.url, err = url.Parse(splunkURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL. %s", err)
	}

	if sw.url.User == nil {
		return nil, fmt.Errorf("missing credentials")
	}

	if sw.url.Path == "" || sw.url.Path[len(sw.url.Path)-1] != '/' {
		sw.url.Path = sw.url.Path + "/"
	}

	sw.username = sw.url.User.Username()
	sw.password, _ = sw.url.User.Password()
	sw.url.User = nil

	setQueryParam := func(ptr *string, key string) {
		values := sw.url.Query()
		if values.Get(key) == "" {
			return
		}
		*ptr = values.Get(key)
		values.Del(key)
		sw.url.RawQuery = values.Encode()
	}
	sw.Index = "default"
	setQueryParam(&sw.Index, "index")
	sw.SourceType = SplunkSourceType
	setQueryParam(&sw.SourceType, "sourcetype")
	sw.Hostname, _ = os.Hostname()
	setQueryParam(&sw.Hostname, "hostname")
	sw.Source = path.Base(os.Args[0])
	setQueryParam(&sw.Source, "source")

	//TODO redo all the formatter stuff so that it's more reusable.
	//     Meaning make it so the splunk template can call the standard event template as a subtemplate.
	sw.Template = template.New("splunk")
	funcMap := template.FuncMap{
		"Hostname": func() string { return sw.Hostname },
		"Source":   func() string { return sw.Source },
		"Pid":      os.Getpid,
	}
	sw.Template.Funcs(funcMap)
	if _, err := sw.Template.Parse(SplunkFormat); err != nil {
		return nil, fmt.Errorf("unable to parse template: %s", err)
	}

	if sw.url.Scheme == "https" {
		var err error
		sw.client, err = getHttpsClient(sw.url.Host)
		if err != nil {
			return nil, err
		}
	} else {
		sw.client = http.DefaultClient
	}

	if _, err := sw.getSessionKey(); err != nil {
		return nil, fmt.Errorf("unable to log in: %s", err)
	}
	return sw, nil
}

// login is responsible for obtaining a new sessionKey from the splunk server.
func (sw *SplunkHandler) login() (string, error) {
	splunkURL, _ := url.Parse(sw.url.String())
	splunkURL.Path = splunkURL.Path + "services/auth/login"

	values := &url.Values{}
	values.Set("username", sw.username)
	values.Set("password", sw.password)
	values.Set("output_mode", "json")

	resp, err := sw.client.PostForm(splunkURL.String(), *values)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("unauthorized")
	}

	buf, _ := ioutil.ReadAll(resp.Body)

	respData := struct{ SessionKey string }{}
	if err := json.Unmarshal(buf, &respData); err != nil {
		return "", err
	}

	if respData.SessionKey == "" {
		return "", fmt.Errorf("failed to obtain sessionKey in response")
	}

	return respData.SessionKey, nil
}

// getSessionKey will return the current session key, or obtain a new one if expired.
func (sw *SplunkHandler) getSessionKey() (string, error) {
	if sw.sessionKey == "" || sw.sessionKeyTime.Before(time.Now().Add(-sessionKeyDuration)) {
		var err error
		sw.sessionKey, err = sw.login()
		if err != nil {
			return "", err
		}

		sw.sessionKeyTime = time.Now()
	}

	return sw.sessionKey, nil
}

// Event processes an event and sends it to the splunk server.
func (sw *SplunkHandler) Event(logEvent *event.Event) error {
	splunkURL, _ := url.Parse(sw.url.String())
	values := splunkURL.Query()
	values.Set("host", sw.Hostname)
	values.Set("source", sw.Source)
	values.Set("sourcetype", sw.SourceType)
	values.Set("index", sw.Index)
	splunkURL.RawQuery = values.Encode()
	splunkURL.Path = splunkURL.Path + "services/receivers/simple"

	sessionKey, err := sw.getSessionKey()
	if err != nil {
		return err
	}

	var templateBuffer bytes.Buffer
	sw.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
	eventBytes := templateBuffer.Bytes()
	eventBytes = bytes.Replace(eventBytes, []byte{'\n'}, []byte{'\r'}, -1)

	req, _ := http.NewRequest("POST", splunkURL.String(), bytes.NewReader(eventBytes))
	req.Header.Set("Authorization", "Splunk "+sessionKey)
	//req.Write(os.Stderr)
	//return nil

	resp, err := sw.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
