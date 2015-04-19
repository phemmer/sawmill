package splunk

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemmer/sawmill/event"
)

func TestNew_http(t *testing.T) {
	sw, err := New(splunkURL(splunkHttpSvr))
	assert.NoError(t, err)
	assert.NotNil(t, sw)
}
func TestNew_https(t *testing.T) {
	sw, err := New(splunkURL(splunkHttpsSvr))
	assert.NoError(t, err)
	assert.NotNil(t, sw)
}

func TestNew_noUser(t *testing.T) {
	u, err := url.Parse(splunkURL(splunkHttpSvr))
	require.NoError(t, err)
	u.User = nil
	_, err = New(u.String())
	assert.Error(t, err)
}

func TestNew_badUser(t *testing.T) {
	u, err := url.Parse(splunkURL(splunkHttpSvr))
	require.NoError(t, err)
	u.User = url.UserPassword("foo", "bar")
	_, err = New(u.String())
	assert.Error(t, err)
}

func TestNew_queryParams(t *testing.T) {
	u, err := url.Parse(splunkURL(splunkHttpSvr))
	require.NoError(t, err)

	values := u.Query()
	values.Set("index", "foo1")
	values.Set("sourcetype", "foo2")
	values.Set("hostname", "foo3")
	values.Set("source", "foo4")
	u.RawQuery = values.Encode()

	sw, err := New(u.String())
	assert.NoError(t, err)

	assert.Equal(t, "foo1", sw.Index)
	assert.Equal(t, "foo2", sw.SourceType)
	assert.Equal(t, "foo3", sw.Hostname)
	assert.Equal(t, "foo4", sw.Source)
}

func TestLogin(t *testing.T) {
	sw, err := New(splunkURL(splunkHttpsSvr))
	require.NoError(t, err)

	key, err := sw.login()
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	valid := splunkSvr.checkToken(key)
	assert.True(t, valid)
}

func TestGetSessionKey(t *testing.T) {
	sw, err := New(splunkURL(splunkHttpsSvr))
	require.NoError(t, err)

	sw.getSessionKey()
	key1 := sw.sessionKey
	require.NotEmpty(t, key1)

	key2, err := sw.getSessionKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, key2)
	assert.Equal(t, key1, key2)
}

func TestGetSessionKey_expired(t *testing.T) {
	sw, err := New(splunkURL(splunkHttpsSvr))
	require.NoError(t, err)

	sw.getSessionKey()
	key1 := sw.sessionKey
	require.NotEmpty(t, key1)

	// mark the key as 1 second past expiration
	sw.sessionKeyTime = time.Now().Add(-(sessionKeyDuration + time.Second))

	key2, err := sw.getSessionKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
}

func TestEvent(t *testing.T) {
	sw, err := New(splunkURL(splunkHttpsSvr))
	require.NoError(t, err)

	logEvent := event.NewEvent(1, event.Warning, "testing Event()", map[string]interface{}{"test": "TestEvent"}, false)
	err = sw.Event(logEvent)
	assert.NoError(t, err)

	serverEvent := splunkSvr.events[len(splunkSvr.events)-1]

	assert.Equal(t, sw.Index, serverEvent.index)
	assert.Equal(t, sw.Hostname, serverEvent.host)
	assert.Equal(t, sw.Source, serverEvent.source)
	assert.Equal(t, sw.SourceType, serverEvent.sourcetype)
	assert.Contains(t, serverEvent.message, logEvent.Message)
	assert.Contains(t, serverEvent.message, "test=TestEvent")
}

func TestNewHttpsClient_splunkCloudHostname(t *testing.T) {
	client, err := newHttpsClient("input-foo.cloud.splunk.com:8089")
	assert.NoError(t, err)
	assert.Equal(t, "foo.cloud.splunk.com", client.Transport.(*http.Transport).TLSClientConfig.ServerName)
}
