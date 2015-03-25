/*
The splunk package is an event handler responsible for sending events to splunk via http stream.
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

const SplunkFormat = "{{.Time \"2006-01-02 15:04:05.000 -0700\"}} {{.Level}}({{.Event.Level}}) {{Source}}[{{Pid}}]: " + formatter.SIMPLE_FORMAT
const SplunkSourceType = "syslog"
const sessionKeyDuration = time.Duration(time.Minute * 15)

type SplunkWriter struct {
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

func NewSplunkWriter(splunkURL string) (*SplunkWriter, error) {
	sw := &SplunkWriter{}

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

func (sw *SplunkWriter) login() (string, error) {
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
func (sw *SplunkWriter) getSessionKey() (string, error) {
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

func (sw *SplunkWriter) Event(logEvent *event.Event) error {
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
