package call

import (
	"github.com/zhifeichen/esl/command"
	"net/textproto"
)

// NoMedia command interface
type NoMedia struct {
	UUID        string
	NoMediaUUID string
	Sync        bool
	SyncPri     bool
}

// BuildMessage Implement command interface
func (n NoMedia) BuildMessage() string {
	sendMsg := command.SendMessage{
		UUID:    n.UUID,
		Headers: make(textproto.MIMEHeader),
		Sync:    n.Sync,
		SyncPri: n.SyncPri,
	}
	sendMsg.Headers.Set("call-command", "nomedia")
	sendMsg.Headers.Set("nomedia-uuid", n.NoMediaUUID)

	return sendMsg.BuildMessage()
}
