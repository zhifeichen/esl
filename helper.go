package esl

import (
	"context"

	"github.com/zhifeichen/esl/v2/command"
	"github.com/zhifeichen/esl/v2/command/call"
)

// EnableEvent subscribe event format and type
func (c *Connection) EnableEvent(ctx context.Context, events ...string) error {
	var err error
	if c.outbound && len(events) == 0 {
		_, err = c.SendCommand(ctx, command.MyEvents{Format: "plain"})
	} else {
		_, err = c.SendCommand(ctx, command.Event{Format: "plain", Listen: events})
	}
	return err
}

// Set execute channel `set` command
func (c *Connection) Set(ctx context.Context, key, value, uuid string) error {
	s := call.Set{
		UUID: uuid,
		Key: key,
		Value: value,
	}
	_, err := c.SendCommand(ctx, s)
	return err
}

// Export execute channel `export` command
func (c *Connection) Export(ctx context.Context, key, value, uuid string) error {
	e := call.Export{
		UUID: uuid,
		Key: key,
		Value: value,
	}
	_, err := c.SendCommand(ctx, e)
	return err
}
