package syslog

import (
  "github.com/phemmer/sawmill/event"
  "github.com/phemmer/sawmill/event/formatter"
	"text/template"
  "os"
  "path"
  "net"
  "fmt"
  "time"
  "bytes"
)

const (
  EMERG int = iota
  ALERT
  CRIT
  ERR
  WARN
  NOTICE
  INFO
  DEBUG
)
const (
  KERN int = iota << 3
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

var levelPriorityMap map[event.Level]int = map[event.Level]int{
  event.Debug: DEBUG,
  event.Info: INFO,
  event.Notice: NOTICE,
  event.Warning: WARN,
  event.Error: ERR,
  event.Critical: CRIT,
  event.Alert: ALERT,
  event.Emergency: EMERG,
}

type SyslogWriter struct {
  syslogProtocol string
  syslogAddr string
  syslogConnection net.Conn
  syslogHostname string
  syslogFacility int
  syslogTag string
  Template *template.Template
}

func NewSyslogWriter(protocol string, addr string, facility int, templateString string) (*SyslogWriter, error) {
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

  sw := &SyslogWriter{
    syslogProtocol: protocol,
    syslogAddr: addr,
    syslogHostname: hostname,
    syslogFacility: facility,
    syslogTag: tag,
    Template: formatterTemplate,
  }

  err = sw.Dial()
  if err != nil {
    return nil, err
  }

  return sw, nil
}

func (sw *SyslogWriter) Dial() (error) {
  if sw.syslogConnection != nil {
    sw.syslogConnection.Close()
    sw.syslogConnection = nil
  }

  // based on log/syslog.Dial()
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

func (sw *SyslogWriter) Event(logEvent *event.Event) (error) {
	var templateBuffer bytes.Buffer
	sw.Template.Execute(&templateBuffer, formatter.EventFormatter(logEvent))
  return sw.sendMessage(logEvent, templateBuffer.Bytes())
}
func (sw *SyslogWriter) sendMessage(event *event.Event, message []byte) (error) {
  priority := sw.syslogFacility | levelPriorityMap[event.Level]
  timestamp := event.Time.Format(time.StampMilli) // this is the BSD syslog format. IETF syslog format is better, but is still relatively new.
  tag := sw.syslogTag
  pid := os.Getpid()

  data := []byte(fmt.Sprintf("<%d>%s %s[%d]: %s\n", priority, timestamp, tag, pid, message))

  if sw.syslogConnection == nil {
    err := sw.Dial()
    if err != nil {
      return err
    }
  }

  _, err := sw.syslogConnection.Write(data)
  if err == nil { // write success
    return nil
  }

  err = sw.Dial()
  if err != nil { // re-dial failed
    return err
  }

  _, err = sw.syslogConnection.Write(data)
  return err
}
