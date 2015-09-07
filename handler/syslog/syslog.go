/*
The syslog package is an event handler responsible for sending events to syslog.
*/
package syslog

import (
	"bytes"
	"fmt"
	"github.com/phemmer/sawmill/event"
	"github.com/phemmer/sawmill/event/formatter"
	"net"
	"os"
	"path"
	"text/template"
	"time"
)

type level int

const (
	EMERG level = iota
	ALERT
	CRIT
	ERR
	WARN
	NOTICE
	INFO
	DEBUG
)

type facility int

const (
	KERN facility = iota << 3
	USER
	MAIL
	DAEMON
	AUTH
	SYSLOG
	LPR
	NEWS
	UUCP
	CRON
	AUTHPRIV
	FTP
	LOCAL0
	LOCAL1
	LOCAL2
	LOCAL3
	LOCAL4
	LOCAL5
	LOCAL6
	LOCAL7
)

var levelPriorityMap map[event.Level]level = map[event.Level]level{
	event.Debug:     DEBUG,
	event.Info:      INFO,
	event.Notice:    NOTICE,
	event.Warning:   WARN,
	event.Error:     ERR,
	event.Critical:  CRIT,
	event.Alert:     ALERT,
	event.Emergency: EMERG,
}

type SyslogHandler struct {
	syslogProtocol   string
	syslogAddr       string
	syslogConnection net.Conn
	syslogHostname   string
	syslogFacility   facility
	syslogTag        string
	Template         *template.Template
}

// New attempts to connect to syslog, and returns a new SyslogHandler if successful.
//
// protocol is a "network" as defined by the net package. Commonly either "unix" or "unixgram". See net.Dial for available values. Defaults to "unix" if emtpy.
//
// addr is the address where to reach the syslog daemon. Also see net.Dial. If empty, "/dev/log", "/var/run/syslog", and "/var/run/log" are tried.
//
// facility is the syslog facility to use for all events processed through this handler. Defaults to USER.
//
// templateString is the sawmill/event/formatter compatable template to use for formatting events. Defaults to formatter.SIMPLE_FORMAT.
func New(protocol string, addr string, facility facility, templateString string) (*SyslogHandler, error) {
	tag := path.Base(os.Args[0])

	if facility == 0 {
		facility = USER
	}

	if templateString == "" {
		templateString = formatter.SIMPLE_FORMAT
	}
	formatterTemplate, err := template.New("").Parse(templateString)
	if err != nil {
		fmt.Printf("Error parsing template: %s", err) //TODO send message somewhere else?
		return nil, err
	}

	hostname, _ := os.Hostname()

	sw := &SyslogHandler{
		syslogProtocol: protocol,
		syslogAddr:     addr,
		syslogHostname: hostname,
		syslogFacility: facility,
		syslogTag:      tag,
		Template:       formatterTemplate,
	}

	err = sw.dial()
	if err != nil {
		return nil, err
	}

	return sw, nil
}

// dial is based on log/syslog.Dial().
// It was copied out as log/syslog.Dial() doesn't properly use the basename of ARGV[0].
func (sw *SyslogHandler) dial() error {
	if sw.syslogProtocol == "" || sw.syslogProtocol == "unix" {
		logTypes := []string{"unixgram", "unix"}
		var logPaths []string
		if sw.syslogAddr != "" {
			logPaths = []string{sw.syslogAddr}
		} else {
			logPaths = []string{"/dev/log", "/var/run/syslog", "/var/run/log"}
		}
		for _, network := range logTypes {
			for _, path := range logPaths {
				conn, err := net.Dial(network, path)
				if err != nil {
					continue
				}
				sw.syslogConnection = conn
				return nil
			}
		}
		return fmt.Errorf("Could not find listening syslog daemon")
	}

	connection, err := net.Dial(sw.syslogProtocol, sw.syslogAddr)
	if err != nil {
		return err
	}
	sw.syslogConnection = connection
	return nil
}

// Event accepts an event and writes it out to the syslog daemon.
// If the connection was lost, the function will attempt to reconnect once.
func (sw *SyslogHandler) Event(logEvent *event.Event) error {
	var templateBuffer bytes.Buffer
	sw.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
	return sw.sendMessage(logEvent, templateBuffer.Bytes())
}

func (sw *SyslogHandler) sendMessage(event *event.Event, message []byte) error {
	priority := int(sw.syslogFacility) | int(levelPriorityMap[event.Level])
	timestamp := event.Time.Format(time.StampMilli) // this is the BSD syslog format. IETF syslog format is better, but is still relatively new.
	tag := sw.syslogTag
	pid := os.Getpid()

	data := []byte(fmt.Sprintf("<%d>%s %s[%d]: %s\n", priority, timestamp, tag, pid, message))

	_, err := sw.syslogConnection.Write(data)
	if err == nil { // write success
		return nil
	}

	err = sw.dial()
	if err != nil { // re-dial failed
		return err
	}

	_, err = sw.syslogConnection.Write(data)
	return err
}
