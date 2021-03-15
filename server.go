package esl

import (
	"context"
	"net"

	"github.com/zhifeichen/log"
)

// OutboundHandler connection handler
type OutboundHandler func(ctx context.Context, conn *Connection)

var listener net.Listener
// ListenAndServe outbound server
func ListenAndServe(addr string, handler OutboundHandler) error {
	var err error
	listener, err = net.Listen("tcp", addr)
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

		handlerCtx, cancel := context.WithCancel(conn.runningContext)
		go conn.dummyLoop(cancel)
		go conn.outboundHandle(handlerCtx, handler)
	}
	log.Info("outbound server shutting down")
	return nil
}

// Shutdown shutdown the outbound server
func Shutdown() {
	listener.Close()
}

func (c *Connection) outboundHandle(ctx context.Context, handler OutboundHandler) {
	handler(ctx, c)
}

func (c *Connection) dummyLoop(cancel context.CancelFunc) {
	for {
		select {
		case e := <-c.responseChns[TypeDisconnect]:
			disposition := e.GetHeader("Content-Disposition")
			if disposition == "linger" {
				log.Info("received linger disconnect...")
				cancel()
				continue
			}
			log.Warnf("Disconnect outbound connection %s\n", c.conn.RemoteAddr().String())
			c.Close()
		case <-c.responseChns[TypeAuthRequest]:
			log.Infof("Ignoring auth request on outbound connectiong %s\n", c.conn.RemoteAddr().String())
		case <-c.runningContext.Done():
			return
		}
	}
}
