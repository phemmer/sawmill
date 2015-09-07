package syslog

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phemmer/sawmill/event"
)

type listener struct {
	netListener net.Listener
	Addr        string
	MsgChan     chan string
}

func newUNIXListener() (*listener, error) {
	addrFunc := func() (string, string) {
		sockPath := path.Join(os.TempDir(), fmt.Sprintf("syslog-%d.sock", rand.Int()))

		return "unix", sockPath
	}
	return newListener(addrFunc)
}

func newTCPListener() (*listener, error) {
	addrFunc := func() (string, string) {
		port := rand.Intn(65535-1024) + 1024
		addr := fmt.Sprintf("127.0.0.1:%d", port)

		return "tcp", addr
	}
	return newListener(addrFunc)
}

func newListener(addrFunc func() (string, string)) (*listener, error) {
	for {
		proto, addr := addrFunc()
		nl, err := net.Listen(proto, addr)
		if err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				continue
			}
			return nil, err
		}

		l := &listener{
			netListener: nl,
			Addr:        addr,
			MsgChan:     make(chan string, 10),
		}
		go l.listen()

		return l, nil
	}
}

func (l *listener) Close() {
	l.netListener.Close()
}

func (l *listener) listen() {
	wg := &sync.WaitGroup{}

	for {
		c, err := l.netListener.Accept()
		if err != nil {
			break
		}
		wg.Add(1)
		go l.serve(c, wg)
	}

	wg.Wait()
	close(l.MsgChan)
}

func (l *listener) serve(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		l.MsgChan <- scanner.Text()
	}
}

func TestNew(t *testing.T) {
	l, err := newUNIXListener()
	require.NoError(t, err)
	defer l.Close()

	handler, err := New("", l.Addr, DAEMON, "")
	require.NoError(t, err)
	assert.NotNil(t, handler)
}

// TestNew_direct tests the code path where the handler does not try and
// determine the path & proto of the unix domain socket.
func TestNew_direct(t *testing.T) {
	l, err := newTCPListener()
	require.NoError(t, err)
	defer l.Close()

	handler, err := New("tcp", l.Addr, DAEMON, "")
	require.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestEvent(t *testing.T) {
	l, err := newUNIXListener()
	require.NoError(t, err)
	defer l.Close()

	handler, err := New("", l.Addr, DAEMON, "")
	require.NoError(t, err)

	logEvent := event.New(1, event.Warning, "testing Event()", map[string]interface{}{"test": "TestEvent"}, false)
	err = handler.Event(logEvent)
	require.NoError(t, err)

	msg := <-l.MsgChan
	assert.Equal(t, "<28>"+logEvent.Time.Format(time.StampMilli)+" syslog.test["+fmt.Sprintf("%d", os.Getpid())+"]: testing Event() -- test=TestEvent", msg)
}

func TestEvent_redial(t *testing.T) {
	l, err := newUNIXListener()
	require.NoError(t, err)
	defer l.Close()

	handler, err := New("", l.Addr, DAEMON, "")
	require.NoError(t, err)

	handler.syslogConnection.Close()

	logEvent := event.New(1, event.Warning, "testing Event()", map[string]interface{}{"test": "TestEvent"}, false)
	err = handler.Event(logEvent)
	require.NoError(t, err)

	msg := <-l.MsgChan
	assert.Equal(t, "<28>"+logEvent.Time.Format(time.StampMilli)+" syslog.test["+fmt.Sprintf("%d", os.Getpid())+"]: testing Event() -- test=TestEvent", msg)
}
