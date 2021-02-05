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
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zhifeichen/log"
)

// Client - In case you need to do inbound dialing against freeswitch server in order to originate call or see
// sofia statuses or whatever else you came up with
type Client struct {
	SocketConnection

	Proto       string `json:"freeswitch_protocol"`
	Addr        string `json:"freeswitch_addr"`
	Passwd      string `json:"freeswitch_password"`
	Timeout     int    `json:"freeswitch_connection_timeout"`
	filter      *filter
	running     bool
	chnClosed   chan struct{}
	eventFormat string
	events      string
}

// EstablishConnection - Will attempt to establish connection against freeswitch and create new SocketConnection
func (c *Client) EstablishConnection() error {
	conn, err := c.Dial(c.Proto, c.Addr, time.Duration(c.Timeout*int(time.Second)))
	if err != nil {
		return err
	}

	c.SocketConnection = SocketConnection{
		Conn: conn,
		err:  make(chan error),
		m:    make(chan *Message),
	}

	return nil
}

// Authenticate - Method used to authenticate client against freeswitch. In case of any errors durring so
// we will return error.
func (c *Client) Authenticate() error {

	m, err := newMessage(bufio.NewReaderSize(c, ReadBufferSize), false)
	if err != nil {
		// log.Error(ECouldNotCreateMessage, err)
		return newEslError("create message error", ErrCouldNotCreateMessage, err)
	}

	cmr, err := m.tr.ReadMIMEHeader()
	if err != nil && err.Error() != "EOF" {
		// log.Error(ECouldNotReadMIMEHeaders, err)
		return newEslError("read mime header error", ErrCouldNotReadMIMEHeaders, err)
	}

	log.Debugf("A: %v\n", cmr)

	if cmr.Get("Content-Type") != "auth/request" {
		// log.Error(EUnexpectedAuthHeader, cmr.Get("Content-Type"))
		return newEslError(cmr.Get("Content-Type"), ErrUnexpectedAuthHeader, nil)
	}

	s := "auth " + c.Passwd + "\r\n\r\n"
	_, err = io.WriteString(c, s)
	if err != nil {
		return err
	}

	am, err := m.tr.ReadMIMEHeader()
	if err != nil && err.Error() != "EOF" {
		// log.Error(ECouldNotReadMIMEHeaders, err)
		return newEslError("", ErrCouldNotReadMIMEHeaders, err)
	}

	if am.Get("Reply-Text") != "+OK accepted" {
		// log.Error(EInvalidPassword, c.Passwd)
		return newEslError(c.Passwd, ErrInvalidPassword, nil)
	}

	return nil
}

// NewClient - Will initiate new client that will establish connection and attempt to authenticate
// against connected freeswitch server
func NewClient(host string, port uint16, passwd string, timeout int) (*Client, error) {
	client := Client{
		Proto:     "tcp", // Let me know if you ever need this open up lol
		Addr:      net.JoinHostPort(host, strconv.Itoa(int(port))),
		Passwd:    passwd,
		Timeout:   timeout,
		running:   false,
		chnClosed: make(chan struct{}),
	}

	filter := filter{
		bgapi:  bgapiFilter{cb: make(map[string]BgapiCallback)},
		event:  eventFilter{cb: make(map[string]EventCallback)},
		header: headerFilter{cb: make([]*headerFilterItem, 0, 5)},
	}
	client.filter = &filter

	return &client, nil
}

// BgapiCallback bgapi callback func
type BgapiCallback func(header map[string]string, body string)

// EventCallback event file callback func
type EventCallback func(eventName string, header map[string]string, body string)

// HeaderFilterCallback filter by header field callback func
type HeaderFilterCallback func(name, value string, header map[string]string, body string)

type bgapiFilter struct {
	sync.Mutex
	cb map[string]BgapiCallback
}

type eventFilter struct {
	sync.RWMutex
	cb map[string]EventCallback
}

type headerFilterItem struct {
	header string
	value  string
	cb     HeaderFilterCallback
}

type headerFilter struct {
	sync.RWMutex
	cb []*headerFilterItem
}

type filter struct {
	bgapi  bgapiFilter
	event  eventFilter
	header headerFilter
}

// BgAPI send bgapi cmd and set callback
func (c *Client) BgAPI(cmd, uuid string, cb BgapiCallback) error {
	c.filter.bgapi.Lock()
	defer c.filter.bgapi.Unlock()

	err := c.SendBgAPIUUID(cmd, uuid)
	if err != nil {
		return err
	}
	c.filter.bgapi.cb[uuid] = cb
	return nil
}

// APIWithTimeout send api cmd and set callback with timeout
func (c *Client) APIWithTimeout(cmd string, duration time.Duration) (map[string]string, string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()

	uuid := uuid.New().String()
	log.Debugf("api with timout, cmd: %s; duration: %d\n", cmd, duration)

	get := make(chan struct{}, 1)
	var resHeader map[string]string
	var resBody string
	c.filter.bgapi.Lock()
	c.filter.bgapi.cb[uuid] = func(header map[string]string, body string) {
		get <- struct{}{}
		resHeader = header
		resBody = body
	}
	c.filter.bgapi.Unlock()

	err := c.SendBgAPIUUID(cmd, uuid)
	if err != nil {
		return nil, "", err
	}

	select {
	case <-get:
		return resHeader, resBody, nil
	case <-ctx.Done():
		c.filter.bgapi.Lock()
		delete(c.filter.bgapi.cb, uuid)
		c.filter.bgapi.Unlock()
		return nil, "", ErrTimeout
	}
}

// SendBgAPI - Helper designed to attach bgapi in front of the command so that you do not need to write it
func (c *Client) SendBgAPI(command string) error {
	return c.Send("bgapi " + command)
}

// FilterEvent add event filter callback
func (c *Client) FilterEvent(name string, cb EventCallback) {
	c.filter.event.Lock()
	defer c.filter.event.Unlock()

	c.filter.event.cb[name] = cb
}

// FilterHeader filter by header name and value, add callback
func (c *Client) FilterHeader(header, value string, cb HeaderFilterCallback) {
	c.filter.header.Lock()
	defer c.filter.header.Unlock()

	for _, hf := range c.filter.header.cb {
		if hf.header == header && hf.value == value {
			hf.cb = cb
			return
		}
	}
	f := headerFilterItem{
		header: header,
		value:  value,
		cb:     cb,
	}
	c.filter.header.cb = append(c.filter.header.cb, &f)
}

// Events send event
func (c *Client) Events(format, events string) error {
	c.eventFormat = format
	c.events = events
	return c.Send(fmt.Sprintf("event %s %s", format, events))
}

func (c *Client) process() {
	go c.Handle()
loop:
	for {
		msg, err := c.ReadMessage()
		if err != nil {
			break loop
		}

		eventName := msg.GetHeader("Event-Name")
		if eventName == "BACKGROUND_JOB" {
			uuid := msg.GetHeader("Job-Uuid")
			func() {
				c.filter.bgapi.Lock()
				defer c.filter.bgapi.Unlock()

				if fn, ok := c.filter.bgapi.cb[uuid]; ok {
					body := string(msg.Body)
					fn(msg.Headers, body)
					delete(c.filter.bgapi.cb, uuid)
				}
			}()
			continue loop
		}

		{
			fn, ok := func() (EventCallback, bool) {
				c.filter.event.RLock()
				defer c.filter.event.RUnlock()

				fn, ok := c.filter.event.cb[eventName]
				return fn, ok
			}()
			if ok {
				fn(eventName, msg.Headers, string(msg.Body))
				continue loop
			}
		}

		{
			f, ok := func() (*headerFilterItem, bool) {
				c.filter.header.RLock()
				defer c.filter.header.RUnlock()

				for _, f := range c.filter.header.cb {
					if v, ok := msg.Headers[f.header]; ok {
						if v == f.value && f.cb != nil {
							return f, true
						}
					}
				}
				return nil, false
			}()
			if ok {
				f.cb(f.header, f.value, msg.Headers, string(msg.Body))
				continue loop
			}
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

		err = c.Authenticate()
		if err != nil {
			c.Close()
			panic("auth failue")
		}

		once.Do(func() {
			connected <- struct{}{}
		})
		c.Send(fmt.Sprintf("event %s %s", c.eventFormat, c.events))
		c.process()
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
	case <-time.After(time.Duration(c.Timeout) * time.Second):
		c.running = false
		c.Close()
		return errors.New("connect timeout")
	}
}

// Stop stop process loop
func (c *Client) Stop() {
	c.running = false
	c.Close()
	<-c.chnClosed
}
