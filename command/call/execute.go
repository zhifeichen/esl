package call

import (
	"fmt"
	"github.com/zhifeichen/esl/command"
	"net/textproto"
	"strconv"
)

// Execute command interface
type Execute struct {
	UUID      string
	AppName   string
	AppArgs   string
	AppUUID   string
	Loops     int
	Sync      bool
	SyncPri   bool
	ForceBody bool
}

// Set Helper to call Execute with Set since it is commonly used
type Set struct {
	UUID    string
	Key     string
	Value   string
	Sync    bool
	SyncPri bool
}

// Export Helper to call Execute with Export since it is commonly used
type Export Set

// Push Helper to call Execute with Push since it is commonly used
type Push Set

func (s Set) buildMessage(app string) string {
	e := Execute{
		UUID:      s.UUID,
		AppName:   app,
		AppArgs:   fmt.Sprintf("%s=%s", s.Key, s.Value),
		Sync:      s.Sync,
		SyncPri:   s.SyncPri,
		ForceBody: true,
	}
	return e.BuildMessage()
}

// BuildMessage Implement command interface
func (s Set) BuildMessage() string {
	return s.buildMessage("set")
}

// BuildMessage Implement command interface
func (e Export) BuildMessage() string {
	return Set(e).buildMessage("export")
}

// BuildMessage Implement command interface
func (p Push) BuildMessage() string {
	return Set(p).buildMessage("push")
}

// BuildMessage Implement command interface
func (e *Execute) BuildMessage() string {
	if e.Loops == 0 {
		e.Loops = 1
	}
	sendMsg := command.SendMessage{
		UUID:    e.UUID,
		Headers: make(textproto.MIMEHeader),
		Sync:    e.Sync,
		SyncPri: e.SyncPri,
	}
	sendMsg.Headers.Set("call-command", "execute")
	sendMsg.Headers.Set("execute-app-name", e.AppName)
	sendMsg.Headers.Set("loops", strconv.Itoa(e.Loops))
	// This allows us to track when application execution completes via the Application-UUID header in events.
	if e.AppUUID != "" {
		sendMsg.Headers.Set("Event-UUID", e.AppUUID)
	}

	// According to documentation that is the max header length
	if len(e.AppArgs) > 2048 || e.ForceBody {
		sendMsg.Headers.Set("content-type", "text/plain")
		sendMsg.Headers.Set("content-length", strconv.Itoa(len(e.AppArgs)))
		sendMsg.Body = e.AppArgs
	} else {
		sendMsg.Headers.Set("execute-app-arg", e.AppArgs)
	}

	return sendMsg.BuildMessage()
}
