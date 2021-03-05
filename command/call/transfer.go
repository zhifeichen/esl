package call

import (
	"github.com/zhifeichen/esl/v2/command"
	"net/textproto"
)

// Transfer Documentation is sparse on this, but it looks like it transfers a call to an application?
type Transfer struct {
	UUID        string
	Application string
	Sync        bool
	SyncPri     bool
}

// BuildMessage Implement command interface
func (t Transfer) BuildMessage() string {
	sendMsg := command.SendMessage{
		UUID:    t.UUID,
		Headers: make(textproto.MIMEHeader),
		Sync:    t.Sync,
		SyncPri: t.SyncPri,
	}
	sendMsg.Headers.Set("call-command", "xferext")
	sendMsg.Headers.Set("application", t.Application)

	return sendMsg.BuildMessage()
}
