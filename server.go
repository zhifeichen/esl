package esl

import (
	"context"
	"net"
	"time"

	"github.com/zhifeichen/esl/v2/command"
	"github.com/zhifeichen/log"
)

// OutboundHandler connection handler
type OutboundHandler func(ctx context.Context, conn *Connection)

// ListenAndServe outbound server
func ListenAndServe(addr string, handler OutboundHandler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("Listenting for new ESL connections on %s\n", listener.Addr().String())
	for {
		c, err := listener.Accept()
		if err != nil {
			log.Error(err)
			break
		}
		log.Infof("New outbound connection from %s\n", c.RemoteAddr().String())
		conn := newConnect(c, true)

		go conn.receiveLoop()
		go conn.eventLoop()

		go conn.dummyLoop()
		go conn.outboundHandle(handler)
	}
	log.Info("outbound server shutting down")
	return nil
}

func (c *Connection) outboundHandle(handler OutboundHandler) {
	// ctx, cancel := context.WithTimeout(c.runningContext, 3 * time.Second)
	// resp, err := c.SendCommand(ctx, command.Connect{})
	// cancel()
	// if err != nil {
	// 	log.Errorf("Error connecting to %s error %s\n", c.conn.RemoteAddr().String(), err.Error())
	// 	c.Close()
	// 	return
	// }

	handler(c.runningContext, c)

	time.Sleep(25 * time.Millisecond)
	ctx, cancel := context.WithTimeout(c.runningContext, 3 * time.Second)
	c.SendCommand(ctx, command.Exit{})
	cancel()
	c.ExitAndClose()
}

func (c *Connection) dummyLoop() {
	select {
	case <- c.responseChns[TypeDisconnect]:
		log.Warnf("Disconnect outbound connection %s\n", c.conn.RemoteAddr().String())
		c.Close()
	case <- c.responseChns[TypeAuthRequest]:
		log.Infof("Ignoring auth request on outbound connectiong %s\n", c.conn.RemoteAddr().String())
	case <- c.runningContext.Done():
		return
	}
}
