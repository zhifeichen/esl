package esl

import (
	"context"
	"net"
)

// OutboundHandler connection handler
type OutboundHandler func(ctx context.Context, conn *Connection)

type outboundServer struct {
	listener net.Listener
	ctx      context.Context
	stop     context.CancelFunc
}

var server outboundServer

// ListenAndServe outbound server
func ListenAndServe(addr string, handler OutboundHandler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error(err)
		return err
	}
	server.listener = listener
	server.ctx, server.stop = context.WithCancel(context.Background())
	logger.Infof("Listenting for new ESL connections on %s\n", listener.Addr().String())
	for {
		c, err := listener.Accept()
		if err != nil {
			logger.Error(err)
			break
		}
		logger.Infof("New outbound connection from %s\n", c.RemoteAddr().String())
		conn := newConnect(server.ctx, c, true)

		go conn.receiveLoop()
		go conn.eventLoop()

		handlerCtx, cancel := context.WithCancel(conn.runningContext)
		go conn.dummyLoop(cancel)
		go conn.outboundHandle(handlerCtx, handler)
	}
	logger.Info("outbound server shutting down")
	server.stop()
	return nil
}

// Shutdown shutdown the outbound server
func Shutdown() {
	server.listener.Close()
	<-server.ctx.Done()
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
				logger.Info("received linger disconnect...")
				cancel()
				continue
			}
			logger.Warnf("Disconnect outbound connection %s\n", c.conn.RemoteAddr().String())
			c.Close()
		case <-c.responseChns[TypeAuthRequest]:
			logger.Infof("Ignoring auth request on outbound connectiong %s\n", c.conn.RemoteAddr().String())
		case <-c.runningContext.Done():
			return
		}
	}
}
