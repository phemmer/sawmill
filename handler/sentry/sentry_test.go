package sentry

import (
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemmer/sawmill/event"
)

func TestSentryHandler(t *testing.T) {
	st := &sentryTransport{
		user:      "0123456789abcdef0123456789abcdef",
		pass:      "fedcba9876543210fedcba9876543210",
		projectID: "12345",
	}

	dsn := fmt.Sprintf("http://%s:%s@%s/%s", "0123456789abcdef0123456789abcdef", "fedcba9876543210fedcba9876543210", "localhost:2", "12345")
	handler, err := New(dsn)
	handler.Tag("abcd", "1234")
	require.NoError(t, err)
	handler.client.Transport = st

	logEvent := event.NewEvent(1, event.Warning, "test SentryHandler", map[string]interface{}{"foo": "bar", "breakfast": map[string]interface{}{"pop": "tart"}}, true)
	err = handler.Event(logEvent)
	require.NoError(t, err)

	packet := st.packets[len(st.packets)-1]
	assert.Equal(t, "test SentryHandler", packet.Message)
	assert.Equal(t, handler.idPrefix+"1", packet.EventID)
	assert.WithinDuration(t, time.Now(), time.Time(packet.Timestamp), time.Second)
	assert.Equal(t, raven.WARNING, packet.Level)
	assert.Equal(t, "sentry.TestSentryHandler", packet.Culprit)
	assert.Equal(t, "sawmill", packet.Logger)
	assert.Equal(t, "go", packet.Platform)
	assert.NotEmpty(t, packet.Release)
	assert.Equal(t, "bar", packet.Extra["foo"])
	assert.Equal(t, "tart", packet.Extra["breakfast.pop"])
	assert.Contains(t, packet.Tags, raven.Tag{Key: "abcd", Value: "1234"})

	haveStacktrace := false
	for _, iface := range packet.Interfaces {
		switch iface := iface.(type) {
		case *raven.Stacktrace:
			haveStacktrace = true
			frame := iface.Frames[len(iface.Frames)-1]
			assert.Equal(t, "handler/sentry/sentry_test.go", frame.Filename)
			assert.Equal(t, "TestSentryHandler", frame.Function)
			assert.Equal(t, "sentry", frame.Module)
			assert.NotEmpty(t, frame.Lineno)
			assert.Equal(t, uint8('/'), frame.AbsolutePath[0])
			assert.Contains(t, frame.AbsolutePath, "/handler/sentry/sentry_test.go")
			assert.Equal(t, true, frame.InApp)
			assert.Equal(t, "	handler.client.Transport = st", frame.PreContext[1])
			assert.Equal(t, `	logEvent := event.NewEvent(1, event.Warning, "test SentryHandler", map[string]interface{}{"foo": "bar", "breakfast": map[string]interface{}{"pop": "tart"}}, true)`, frame.ContextLine)
		}
	}
	assert.True(t, haveStacktrace)

	handler.Stop()
}
