// Copyright 2020 chen zhifei
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package esl

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/textproto"
	"strconv"
	"sync"
	"time"

	"github.com/zhifeichen/esl/v2/command"
	"github.com/zhifeichen/log"
)

// Client - In case you need to do inbound dialing against freeswitch server in order to originate call or see
// sofia statuses or whatever else you came up with
type Client struct {
	Connection

	Proto   string `json:"freeswitch_protocol"`
	Addr    string `json:"freeswitch_addr"`
	Passwd  string `json:"freeswitch_password"`
	Timeout int    `json:"freeswitch_connection_timeout"`

	running      bool
	chnClosed    chan struct{}
	eventFormat  string
	events       string
	sendConnCnt  int
	sendConn     []*Connection
	sendParamChn chan *sendParam
}

type sendParam struct {
	ctx context.Context
	cmd command.Command
	fn  []EventHandler
}

// NewClient - Will initiate new client that will establish connection and attempt to authenticate
// against connected freeswitch server
func NewClient(host string, port uint16, passwd string, timeout, sendConnCnt int) (*Client, error) {
	client := Client{
		Proto:        "tcp", // Let me know if you ever need this open up lol
		Addr:         net.JoinHostPort(host, strconv.Itoa(int(port))),
		Passwd:       passwd,
		Timeout:      timeout,
		running:      false,
		chnClosed:    make(chan struct{}, 1),
		sendParamChn: make(chan *sendParam),
		sendConnCnt:  sendConnCnt,
		sendConn:     make([]*Connection, sendConnCnt),
	}

	return &client, nil
}

// EstablishConnection - Will attempt to establish connection against freeswitch and create new SocketConnection
func (c *Client) EstablishConnection() error {

	runningCtx, stop := context.WithCancel(context.Background())

	// save orig filter
	var origFilter *filter
	if c.Connection.filter != nil {
		origFilter = c.Connection.filter
	}

	c.Connection = Connection{
		runningContext: runningCtx,
		stop:           stop,
		outbound:       false,
		responseChns: map[string]chan *RawResponse{
			TypeReply:       make(chan *RawResponse),
			TypeAPIResponse: make(chan *RawResponse),
			TypeEventPlain:  make(chan *RawResponse),
			TypeEventJSON:   make(chan *RawResponse),
			TypeEventXML:    make(chan *RawResponse),
			TypeAuthRequest: make(chan *RawResponse),
			TypeDisconnect:  make(chan *RawResponse),
		},
	}
	if origFilter != nil {
		c.Connection.filter = origFilter
	} else {
		filter := &filter{
			bgapi:  bgFilter{cb: make(map[string]EventHandler)},
			event:  eventFilter{cb: make(map[string]EventHandler)},
			header: headerFilter{cb: make([]*headerFilterItem, 0, 5)},
		}
		c.Connection.filter = filter
	}

	log.Debugf("dial to %s %s...\n", c.Proto, c.Addr)
	to := time.Duration(c.Timeout * int(time.Second))
	conn, err := c.Dial(c.Proto, c.Addr, to)
	if err != nil {
		log.Error(err)
		return err
	}

	c.Connection.conn = conn
	c.Connection.reader = bufio.NewReader(conn)
	c.Connection.header = textproto.NewReader(c.Connection.reader)

	for i, sc := range c.sendConn {
		sconn, err := sc.Dial(c.Proto, c.Addr, to)
		if err != nil {
			log.Error(err)
			c.Close()
			return err
		}
		c.sendConn[i] = newConnect(sconn, false)
	}

	log.Infof("connect to %s success\n", conn.RemoteAddr().String())

	return nil
}

// DoAuth authenticate client against freeswitch.
func (c *Client) DoAuth(ctx context.Context, auth command.Auth) error {
	response, err := c.SendCommand(ctx, auth)
	if err != nil {
		return err
	}
	if !response.IsOk() {
		return ErrInvalidPassword
	}
	return nil
}

func (c *Connection) sendLoop(chn <-chan *sendParam) {
	for cd := range chn {
		resp, err := c.SendCommand(cd.ctx, cd.cmd, cd.fn...)
		if err != nil {
			log.Error(err)
			continue
		}
		if len(cd.fn) > 0 {
			e := &Event{
				Headers: make(textproto.MIMEHeader),
			}
			for k, v := range resp.Headers {
				for _, vv := range v {
					e.Headers.Add(k, vv)
				}
			}
			copy(e.Body, resp.Body)
			cd.fn[0](e)
		}
	}
}

func (c *Connection) runningLoop(passwd string) {
	for {
		select {
		case <-c.runningContext.Done():
			c.Close()
			return
		case <-c.responseChns[TypeAuthRequest]:
			auth := command.Auth{Passwd: passwd}
			c.SendCommand(c.runningContext, auth)
			// c.EnableEvent(c.runningContext, "BACKGROUND_JOB")
		case <-c.responseChns[TypeDisconnect]:
			c.Close()
			return
		}
	}
}

func (c *Client) loop(connected chan<- struct{}) {
	var once sync.Once
	for c.running {
		err := c.EstablishConnection()
		if err != nil {
			<-time.After(2 * time.Second)
			continue
		}

		go c.receiveLoop()
		go c.eventLoop()

		<-c.responseChns[TypeAuthRequest]
		err = c.DoAuth(c.runningContext, command.Auth{Passwd: c.Passwd})
		if err != nil {
			c.Close()
			panic("auth failue")
		}

		once.Do(func() {
			connected <- struct{}{}
		})
		c.EnableEvent(c.runningContext, c.events)
		for _, sc := range c.sendConn {
			go sc.runningLoop(c.Passwd)
			go sc.receiveLoop()
			go sc.eventLoop()

			go sc.sendLoop(c.sendParamChn)
		}

		select {
		case <-c.responseChns[TypeAuthRequest]:
			err := c.DoAuth(c.runningContext, command.Auth{Passwd: c.Passwd})
			if err != nil {
				log.Errorf("authenticate %s error: %s\n", c.conn.RemoteAddr(), err.Error())
				c.ExitAndClose()
				return
			}
			log.Infof("successfully authenticated %s\n", c.conn.RemoteAddr())
		case <-c.responseChns[TypeDisconnect]:
			c.Close()
			log.Warnf("connection disconnected\n")
			continue
		case <-c.runningContext.Done():
			log.Debug("run context done")
			c.chnClosed <- struct{}{}
			return
		}
	}
	c.chnClosed <- struct{}{}
}

// Start start process loop
func (c *Client) Start(format, events string) error {
	if c.running {
		return nil
	}
	c.running = true
	connected := make(chan struct{})
	c.eventFormat = format
	c.events = events
	go c.loop(connected)
	select {
	case <-connected:
		return nil
	case <-time.After(time.Duration(c.Timeout*2) * time.Second):
		c.running = false
		// c.Close()
		return errors.New("connect timeout")
	}
}

// Stop stop process loop
func (c *Client) Stop() {
	c.running = false
	c.Close()
	close(c.sendParamChn)
	log.Info("close done")
	<-c.chnClosed
	log.Info("done")
}

func (c *Client) SendCommand2(ctx context.Context, cmd command.Command, fn ...EventHandler) {
	cd := sendParam{
		ctx: ctx,
		cmd: cmd,
		fn: make([]EventHandler, 0),
	}
	cd.fn = append(cd.fn, fn...)
	c.sendParamChn <- &cd
}
