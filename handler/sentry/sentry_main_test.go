package sentry

import (
	"fmt"
	"strings"

	"github.com/getsentry/raven-go"

	"github.com/phemmer/sawmill/event"
)

func init() {
	// so that the event stack trace won't ignore us because we're in the sawmill
	// repo.
	event.RepoPath = event.FilePath
}

type sentryTransport struct {
	user      string
	pass      string
	projectID string
	packets   []*raven.Packet
}

func (st *sentryTransport) Send(url string, authHeader string, packet *raven.Packet) error {
	if !strings.Contains(authHeader, fmt.Sprintf("sentry_key=%s", st.user)) {
		return fmt.Errorf("unauthorized")
	}
	if !strings.Contains(authHeader, fmt.Sprintf("sentry_secret=%s", st.pass)) {
		return fmt.Errorf("unauthorized")
	}

	if !strings.Contains(url, fmt.Sprintf("/api/%s/store", st.projectID)) {
		return fmt.Errorf("unauthorized")
	}

	st.packets = append(st.packets, packet)
	return nil
}
