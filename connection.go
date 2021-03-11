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
	"sync"
	"time"

	"github.com/zhifeichen/esl/v2/command"
	"github.com/zhifeichen/log"
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

func newConnect(c net.Conn, outbound bool) *Connection {
	reader := bufio.NewReader(c)
	header := textproto.NewReader(reader)

	runningCtx, stop := context.WithCancel(context.Background())

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

	for key, chn := range c.responseChns {
		close(chn)
		delete(c.responseChns, key)
	}

	c.conn.Close()
}

// SendCommand send command to fs
func (c *Connection) SendCommand(ctx context.Context, cmd command.Command, fn ...EventHandler) (*RawResponse, error) {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	log.Debugf("send command: %s\n", cmd.BuildMessage())

	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetWriteDeadline(deadline)
	}
	_, err := c.conn.Write([]byte(cmd.BuildMessage() + EndOfMessage))
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
	}
}

func (c *Connection) receiveLoop() {
	for c.runningContext.Err() == nil {
		err := c.doReceive()
		if err != nil {
			log.Errorf("Error receiving message: %s\n", err.Error())
			break
		}
	}
}

func (c *Connection) doReceive() error {
	response, err := c.readResponse()
	if err != nil {
		return err
	}

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
			// Do not return an error since this is not fatal but log since it could be a indication of problems
			log.Warnf("No one to handle response\nIs the connection overloaded or stopping?\n%v\n\n", response)
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
			log.Errorf("Error parsing event\n%s\n", err.Error())
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

/*
// Send - Will send raw message to open net connection
func (c *Connection) Send(cmd string) error {

	if strings.Contains(cmd, "\r\n") {
		return newEslError(cmd, ErrInvalidCommandProvided, nil)
	}

	// lock mutex
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, cmd)
	if err != nil {
		return err
	}

	_, err = io.WriteString(c, "\r\n\r\n")
	if err != nil {
		return err
	}

	return nil
}

// SendBgAPIUUID - Will send raw message to open net connection
func (c *Connection) SendBgAPIUUID(cmd, uuid string) error {

	if strings.Contains(cmd, "\r\n") {
		return newEslError(cmd, ErrInvalidCommandProvided, nil)
	}

	if strings.Contains(uuid, "\r\n") {
		return newEslError(cmd, ErrInvalidCommandProvided, nil)
	}

	// lock mutex
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, "bgapi "+cmd)
	if err != nil {
		return err
	}

	_, err = io.WriteString(c, "\r\nJob-UUID: ")
	if err != nil {
		return err
	}

	_, err = io.WriteString(c, uuid)
	if err != nil {
		return err
	}

	_, err = io.WriteString(c, "\r\n\r\n")
	if err != nil {
		return err
	}

	return nil
}

// SendMany - Will loop against passed commands and return 1st error if error happens
func (c *Connection) SendMany(cmds []string) error {

	for _, cmd := range cmds {
		if err := c.Send(cmd); err != nil {
			return err
		}
	}

	return nil
}

// SendEvent - Will loop against passed event headers
func (c *Connection) SendEvent(eventHeaders []string) error {
	if len(eventHeaders) <= 0 {
		return newEslError("", ErrCouldNotSendEvent, nil)
	}

	// lock mutex to prevent event headers from conflicting
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, "sendevent ")
	if err != nil {
		return err
	}

	for _, eventHeader := range eventHeaders {
		_, err := io.WriteString(c, eventHeader)
		if err != nil {
			return err
		}

		_, err = io.WriteString(c, "\r\n")
		if err != nil {
			return err
		}

	}

	_, err = io.WriteString(c, "\r\n")
	if err != nil {
		return err
	}

	return nil
}

// SendEventWithData - Will loop against passed event headers
func (c *Connection) SendEventWithData(event string, eventHeaders map[string]string, data string) error {
	if len(eventHeaders) <= 0 {
		return newEslError("", ErrCouldNotSendEvent, nil)
	}

	// lock mutex to prevent event headers from conflicting
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, fmt.Sprintf("sendevent %s\r\n", event))
	if err != nil {
		return err
	}

	for key, eventHeader := range eventHeaders {
		_, err := io.WriteString(c, fmt.Sprintf("%s: %s\r\n", key, eventHeader))
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(c, "\r\n")
	if err != nil {
		return err
	}

	if eventHeaders["content-length"] != "" && data != "" {
		io.WriteString(c, data)
	}

	return nil
}

// Execute - Helper fuck to execute commands with its args and sync/async mode
func (c *Connection) Execute(command, args string, sync bool) (m *Message, err error) {
	return c.SendMsg(map[string]string{
		"call-command":     "execute",
		"execute-app-name": command,
		"execute-app-arg":  args,
		"event-lock":       strconv.FormatBool(sync),
	}, "", "")
}

// ExecuteUUID - Helper fuck to execute uuid specific commands with its args and sync/async mode
func (c *Connection) ExecuteUUID(uuid string, command string, args string, sync bool) (m *Message, err error) {
	return c.SendMsg(map[string]string{
		"call-command":     "execute",
		"execute-app-name": command,
		"execute-app-arg":  args,
		"event-lock":       strconv.FormatBool(sync),
	}, uuid, "")
}

// SendMsg - Basically this func will send message to the opened connection
func (c *Connection) SendMsg(msg map[string]string, uuid, data string) (m *Message, err error) {

	b := bytes.NewBufferString("sendmsg")

	if uuid != "" {
		if strings.Contains(uuid, "\r\n") {
			return nil, newEslError(uuid, ErrInvalidCommandProvided, nil)
		}

		b.WriteString(" " + uuid)
	}

	b.WriteString("\r\n")

	for k, v := range msg {
		if strings.Contains(k, "\r\n") {
			return nil, newEslError(k, ErrInvalidCommandProvided, nil)
		}

		if v != "" {
			if strings.Contains(v, "\r\n") {
				return nil, newEslError(v, ErrInvalidCommandProvided, nil)
			}

			b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}

	b.WriteString("\r\n")

	if msg["content-length"] != "" && data != "" {
		b.WriteString(data)
	}

	// lock mutex
	c.mtx.Lock()
	_, err = b.WriteTo(c)
	if err != nil {
		c.mtx.Unlock()
		return nil, err
	}
	c.mtx.Unlock()

	select {
	case err := <-c.err:
		return nil, err
	case m := <-c.m:
		return m, nil
	}
}

// OriginatorAddr - Will return originator address known as net.RemoteAddr()
// This will actually be a freeswitch address
func (c *Connection) OriginatorAddr() net.Addr {
	return c.RemoteAddr()
}

// ReadMessage - Will read message from channels and return them back accordingy.
// If error is received, error will be returned. If not, message will be returned back!
func (c *Connection) ReadMessage() (*Message, error) {
	// log.Debug("Waiting for connection message to be received ...")

	select {
	case err := <-c.err:
		return nil, err
	case msg := <-c.m:
		return msg, nil
	}
}

// Handle - Will handle new messages and close connection when there are no messages left to process
func (c *Connection) Handle() {

	done := make(chan bool)

	rbuf := bufio.NewReaderSize(c, ReadBufferSize)

	go func() {
		for {
			msg, err := newMessage(rbuf, true)

			if err != nil {
				c.err <- err
				done <- true
				break
			}

			c.m <- msg
		}
	}()

	<-done

	// Closing the connection now as there's nothing left to do ...
	c.Close()
}

// Close - Will close down net connection and return error if error happen
func (c *Connection) Close() error {
	if err := c.Conn.Close(); err != nil {
		return err
	}

	return nil
}
*/
