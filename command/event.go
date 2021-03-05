package command

import (
	"fmt"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
)

// Event event command
type Event struct {
	Ignore bool
	Format string
	Listen []string
}

// MyEvents myevents
type MyEvents struct {
	Format string
	UUID string
}

// DisableEvents disable events
type DisableEvents struct {}

// DivertEvents The divert_events command is available to allow events that an embedded script would expect to get in the inputcallback to be diverted to the event socket.
type DivertEvents struct {
	Enabled bool
}

// SendEvent send event
type SendEvent struct {
	Name    string
	Headers textproto.MIMEHeader
	Body    string
}

// BuildMessage Implement command interface
func (e Event) BuildMessage() string {
	prefix := ""
	if e.Ignore {
		prefix = "nix"
	}
	return fmt.Sprintf("%sevent %s %s", prefix, e.Format, strings.Join(e.Listen, " "))
}

// BuildMessage Implement command interface
func (m MyEvents) BuildMessage() string {
	if len(m.UUID) > 0 {
		return fmt.Sprintf("myevents %s %s", m.Format, m.UUID)

	}
	return fmt.Sprintf("myevents %s", m.Format)
}

// BuildMessage Implement command interface
func (DisableEvents) BuildMessage() string {
	return "noevents"
}

// BuildMessage Implement command interface
func (d DivertEvents) BuildMessage() string {
	if d.Enabled {
		return "divert_events on"
	}
	return "divert_events off"
}

// BuildMessage Implement command interface
func (s *SendEvent) BuildMessage() string {
	// Ensure the correct content length is set in the header
	if len(s.Body) > 0 {
		s.Headers.Set("Content-Length", strconv.Itoa(len(s.Body)))
	} else {
		delete(s.Headers, "Content-Length")
	}

	// Format the headers
	var headers strings.Builder
	err := http.Header(s.Headers).Write(&headers)
	if err != nil || headers.Len() < 3 {
		return ""
	}
	// -2 to remove the trailing \r\n added by http.Header.Write
	headerString := headers.String()[:headers.Len()-2]
	if _, ok := s.Headers["Content-Length"]; ok {
		return fmt.Sprintf("sendevent %s\r\n%s\r\n\r\n%s", s.Name, headerString, s.Body)
	}
	return fmt.Sprintf("sendevent %s\r\n%s", s.Name, headerString)
}
