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
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/zhifeichen/esl/v2/command"
)

// Connection Main connection against ESL - Gotta add more description here
type Connection struct {
	conn           net.Conn
	reader         *bufio.Reader
	header         *textproto.Reader
	writeLock      sync.Mutex
	runningContext context.Context
	stop           func()
	responseChns   map[string]chan *RawResponse
	responseChnMtx sync.RWMutex
	filter         *filter
	filterMtx      sync.RWMutex
	outbound       bool
	closeOnce      sync.Once
}

// Dial - Will establish timedout dial against specified address. In this case, it will be freeswitch server
func (c *Connection) Dial(network string, addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func newConnect(ctx context.Context, c net.Conn, outbound bool) *Connection {
	reader := bufio.NewReader(c)
	header := textproto.NewReader(reader)

	runningCtx, stop := context.WithCancel(ctx)

	instance := &Connection{
		conn:           c,
		reader:         reader,
		header:         header,
		runningContext: runningCtx,
		stop:           stop,
		outbound:       outbound,
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
	filter := &filter{
		bgapi:  bgFilter{cb: make(map[string]EventHandler)},
		event:  eventFilter{cb: make(map[string]EventHandler)},
		header: headerFilter{cb: make([]*headerFilterItem, 0, 5)},
	}
	instance.filter = filter
	return instance
}

// RemoteAddr return connection remote addr
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// LocalAddr return connection local addr
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// ExitAndClose send exit and close connection
func (c *Connection) ExitAndClose() {
	c.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(c.runningContext, time.Second)
		c.SendCommand(ctx, command.Exit{})
		cancel()
		c.close()
	})
}

// Close close connection
func (c *Connection) Close() {
	c.closeOnce.Do(c.close)
}

func (c *Connection) close() {
	c.stop()
	c.responseChnMtx.Lock()
	defer c.responseChnMtx.Unlock()

	logger.Info("close")
	for key, chn := range c.responseChns {
		close(chn)
		delete(c.responseChns, key)
		logger.Infof("delete chn %s\n", key)
	}

	logger.Info("close conn")
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	logger.Info("closed")
}

// SendCommand send command to fs
func (c *Connection) SendCommand(ctx context.Context, cmd command.Command, fn ...EventHandler) (*RawResponse, error) {
	t1 := time.Now()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	esc := time.Since(t1).Milliseconds()
	if esc > 500 {
		logger.Errorf("esc > 500: %d\n", esc)
	}

	sendString := cmd.BuildMessage()
	logger.Debugf("send command: %s\n", sendString)
	if c.conn == nil {
		logger.Errorf("send command %s error: Connection closed", sendString)
		return nil, ErrConnClosed
	}

	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetWriteDeadline(deadline)
	}
	_, err := c.conn.Write([]byte(sendString + EndOfMessage))
	if err != nil {
		return nil, err
	}

	background := false
	var cb EventHandler
	if bgCmd, ok := cmd.(command.API); ok && bgCmd.Background && len(fn) > 0 {
		background = true
		cb = fn[len(fn)-1]
	}

	c.responseChnMtx.RLock()
	defer c.responseChnMtx.RUnlock()
	select {
	case response := <-c.responseChns[TypeReply]:
		if response == nil {
			return nil, ErrConnClosed
		}
		if response.IsOk() && background {
			jobid := response.Headers.Get("Job-Uuid")
			if len(jobid) > 0 {
				c.filter.bgapi.Lock()
				c.filter.bgapi.cb[jobid] = cb
				c.filter.bgapi.Unlock()
			}
		}
		return response, nil
	case response := <-c.responseChns[TypeAPIResponse]:
		if response == nil {
			return nil, ErrConnClosed
		}
		if response.IsOk() && background {
			jobid := response.Headers.Get("Job-Uuid")
			if len(jobid) > 0 {
				c.filter.bgapi.Lock()
				c.filter.bgapi.cb[jobid] = cb
				c.filter.bgapi.Unlock()
			}
		}
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.runningContext.Done():
		return nil, c.runningContext.Err()
	}
}

func (c *Connection) receiveLoop() {
	for c.runningContext.Err() == nil {
		err := c.doReceive()
		if err != nil {
			logger.Errorf("Error receiving message: %s; %#v\n", err.Error(), err)
			if strings.Contains(err.Error(), "EOF") {
				logger.Error("connection eof")
				response := RawResponse{
					Headers: make(textproto.MIMEHeader),
				}
				chnName := "text/disconnect-notice"
				response.Headers.Add("Content-Type", chnName)
				c.responseChnMtx.RLock()
				defer c.responseChnMtx.RUnlock()
				// c.responseChns[chnName] <- &response
				select{
				case c.responseChns[chnName] <- &response:
					return
				case <-c.runningContext.Done():
					return
				}
			}
			break
		}
	}
}

func (c *Connection) doReceive() error {
	response, err := c.readResponse()
	if err != nil {
		return err
	}
	logger.Debugf("recv response: %#v\n", response)

	c.responseChnMtx.RLock()
	defer c.responseChnMtx.RUnlock()
	responseChan, ok := c.responseChns[response.GetHeader("Content-Type")]
	if !ok && len(c.responseChns) <= 0 {
		// We must have shutdown!
		return ErrResponseChn
	}

	// We have a handler
	if ok {
		// Only allow 5 seconds to allow the handler to receive hte message on the channel
		ctx, cancel := context.WithTimeout(c.runningContext, 5*time.Second)
		defer cancel()

		select {
		case responseChan <- response:
		case <-c.runningContext.Done():
			// Parent connection context has stopped we most likely shutdown in the middle of waiting for a handler to handle the message
			return c.runningContext.Err()
		case <-ctx.Done():
			// Do not return an error since this is not fatal but logger since it could be a indication of problems
			logger.Warnf("No one to handle response\nIs the connection overloaded or stopping?\n%v\n\n", response)
		}
	} else {
		return errors.New("no response channel for Content-Type: " + response.GetHeader("Content-Type"))
	}
	return nil
}

func (c *Connection) eventLoop() {
	for {
		var event *Event
		var err error
		c.responseChnMtx.RLock()
		select {
		case raw := <-c.responseChns[TypeEventPlain]:
			if raw == nil {
				// We only get nil here if the channel is closed
				c.responseChnMtx.RUnlock()
				return
			}
			event, err = readPlainEvent(raw.Body)
		case raw := <-c.responseChns[TypeEventXML]:
			if raw == nil {
				// We only get nil here if the channel is closed
				c.responseChnMtx.RUnlock()
				return
			}
			event, err = readXMLEvent(raw.Body)
		case raw := <-c.responseChns[TypeEventJSON]:
			if raw == nil {
				// We only get nil here if the channel is closed
				c.responseChnMtx.RUnlock()
				return
			}
			event, err = readJSONEvent(raw.Body)
		case <-c.runningContext.Done():
			c.responseChnMtx.RUnlock()
			return
		}
		c.responseChnMtx.RUnlock()

		if err != nil {
			logger.Errorf("Error parsing event\n%s\n", err.Error())
			continue
		}

		c.handleEvent(event)
	}
}

func (c *Connection) handleEvent(event *Event) {
	c.filterMtx.RLock()
	defer c.filterMtx.RUnlock()

	eventName := event.GetName()
	// first, call background job function
	if eventName == "BACKGROUND_JOB" {
		uuid := event.GetHeader("Job-Uuid")
		func() {
			c.filter.bgapi.Lock()
			defer c.filter.bgapi.Unlock()

			if fn, ok := c.filter.bgapi.cb[uuid]; ok {
				fn(event)
				delete(c.filter.bgapi.cb, uuid)
			}
		}()
		return
	}

	{ // call filter by event name
		fn, ok := func() (EventHandler, bool) {
			c.filter.event.RLock()
			defer c.filter.event.RUnlock()

			fn, ok := c.filter.event.cb[eventName]
			return fn, ok
		}()
		if ok {
			fn(event)
			return
		}
	}

	{ // call filter by header value
		f, ok := func() (*headerFilterItem, bool) {
			c.filter.header.RLock()
			defer c.filter.header.RUnlock()

			for _, f := range c.filter.header.cb {
				v := event.Headers.Values(f.header)
				for _, vv := range v {
					uv, _ := url.PathUnescape(vv)
					if uv == f.value && f.cb != nil {
						return f, true
					}
				}
			}
			return nil, false
		}()
		if ok {
			f.cb(event)
			return
		}
	}

	{ // call ALL handler
		c.filter.event.RLock()
		if fn, ok := c.filter.event.cb[EventListenAll]; ok {
			fn(event)
		}
		c.filter.event.RUnlock()
	}
}

// FilterEvent add event filter callback
func (c *Connection) FilterEvent(name string, cb EventHandler) {
	c.filter.event.Lock()
	defer c.filter.event.Unlock()

	c.filter.event.cb[name] = cb
}

// FilterHeader filter by header name and value, add callback
func (c *Connection) FilterHeader(header, value string, cb EventHandler) {
	c.filter.header.Lock()
	defer c.filter.header.Unlock()

	memeHeader := textproto.CanonicalMIMEHeaderKey(header)
	for _, hf := range c.filter.header.cb {
		if hf.header == memeHeader && hf.value == value {
			hf.cb = cb
			return
		}
	}
	f := headerFilterItem{
		header: memeHeader,
		value:  value,
		cb:     cb,
	}
	c.filter.header.cb = append(c.filter.header.cb, &f)
}
