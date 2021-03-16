// Copyright 2020 chen zhifei
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package esl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/zhifeichen/log"
)

// Message - Freeswitch Message that is received by GoESL. Message struct is here to help with parsing message
// and dumping its contents. In addition to that it's here to make sure received message is in fact message we wish/can support
type Message struct {
	Headers map[string]string
	Body    []byte

	r  *bufio.Reader
	tr *textproto.Reader
}

// EndOfMessage message end
const EndOfMessage = "\r\n\r\n"

// String - Will return message representation as string
func (m Message) String() string {
	var builder strings.Builder
	for key, values := range m.Headers {
		builder.WriteString(fmt.Sprintf("%s: %#v\n", key, values))
	}
	builder.Write(m.Body)
	return builder.String()
}

// GoString implement the GoStringer interface for pretty printing (%#v)
func (m Message) GoString() string {
	return m.String()
}

// GetCallUUID - Will return Caller-Unique-Id
func (m *Message) GetCallUUID() string {
	return m.GetHeader("Caller-Unique-Id")
}

// GetHeader - Will return message header value, or "" if the key is not set.
func (m *Message) GetHeader(key string) string {
	return m.Headers[key]
}

// Parse - Will parse out message received from Freeswitch and basically build it accordingly for later use.
// However, in case of any issues func will return error.
func (m *Message) Parse() error {

	cmr, err := m.tr.ReadMIMEHeader()

	if err != nil && err.Error() != "EOF" {
		// log.Error(ECouldNotReadMIMEHeaders, err)
		return newEslError("57", ErrCouldNotReadMIMEHeaders, err)
	}

	if cmr.Get("Content-Type") == "" {
		log.Debug("Not accepting message because of empty content type. Just whatever with it ...")
		return fmt.Errorf("Parse EOF")
	}

	// Will handle content length by checking if appropriate lenght is here and if it is than
	// we are going to read it into body
	if lv := cmr.Get("Content-Length"); lv != "" {
		l, err := strconv.Atoi(lv)

		if err != nil {
			// log.Error(EInvalidContentLength, err)
			return newEslError("72", ErrInvalidContentLength, err)
		}

		m.Body = make([]byte, l)

		if r, err := io.ReadFull(m.r, m.Body); err != nil {
			// log.Error(ECouldNotReadyBody, err)
			return newEslError(fmt.Sprintf("%d", r), ErrCouldNotReadBody, err)
		}
	}

	msgType := cmr.Get("Content-Type")

	// log.Debugf("Got message content (type: %s). Searching if we can handle it ...", msgType)

	if !StringInSlice(msgType, AvailableMessageTypes) {
		msg := fmt.Sprintf("got: %s, supported types are: %s", msgType, AvailableMessageTypes)
		return newEslError(msg, ErrUnsupportedMessageType, nil)
	}

	// Assing message headers IF message is not type of event-json
	if msgType != "text/event-json" {
		for k, v := range cmr {

			m.Headers[k] = v[0]

			// Will attempt to decode if % is discovered within the string itself
			if strings.Contains(v[0], "%") {
				m.Headers[k], err = url.QueryUnescape(v[0])

				if err != nil {
					log.Error(ErrCouldNotDecode, err)
					continue
				}
			}
		}
	}

	switch msgType {
	case "text/disconnect-notice":
		for k, v := range cmr {
			log.Debugf("Message (header: %s) -> (value: %v)", k, v)
		}
	case "command/reply":
		reply := cmr.Get("Reply-Text")

		if strings.Contains(reply, "-ERR") {
			return fmt.Errorf("%s: %w", reply[5:], ErrUnsuccessfulReply)
		}
	case "api/response":
		if strings.Contains(string(m.Body), "-ERR") {
			return fmt.Errorf("%s: %w", string(m.Body)[5:], ErrUnsuccessfulReply)
		}
	case "text/event-json":
		// OK, what is missing here is a way to interpret other JSON types - it expects string only (understandably
		// because the FS events are generally "string: string") - extract into empty interface and migrate only strings.
		// i.e. Event CHANNEL_EXECUTE_COMPLETE - "variable_DP_MATCH":["a=rtpmap:101 telephone-event/8000","101"]
		var decoded map[string]interface{}

		if err := json.Unmarshal(m.Body, &decoded); err != nil {
			return err
		}

		// Copy back in:
		for k, v := range decoded {
			switch v := v.(type) {
			case string:
				m.Headers[textproto.CanonicalMIMEHeaderKey(k)] = v
			case []string:
				m.Headers[textproto.CanonicalMIMEHeaderKey(k)] = v[0]
			default:
				//delete(m.Headers, k)
				log.Warnf("Removed non-string property (%s)", k)
			}
		}

		if v := m.Headers["_body"]; v != "" {
			m.Body = []byte(v)
			delete(m.Headers, "_body")
		} else {
			m.Body = []byte("")
		}

	case "text/event-plain":
		r := bufio.NewReader(bytes.NewReader(m.Body))

		tr := textproto.NewReader(r)

		emh, err := tr.ReadMIMEHeader()

		if err != nil {
			return newEslError("", ErrCouldNotReadMIMEHeaders, err)
		}
		for k, v := range emh {
			if k != "Content-Length" {
				m.Headers[k] = v[0]
				if strings.Contains(v[0], "%") {
					m.Headers[k], err = url.QueryUnescape(v[0])

					if err != nil {
						log.Error(ErrCouldNotDecode, err)
						continue
					}
				}
			}
		}

		if vl := emh.Get("Content-Length"); vl != "" {
			length, err := strconv.Atoi(vl)

			if err != nil {
				// log.Error(EInvalidContentLength, err)
				return newEslError("", ErrInvalidContentLength, err)
			}

			m.Body = make([]byte, length)

			if _, err = io.ReadFull(r, m.Body); err != nil {
				// log.Error(ECouldNotReadyBody, err)
				return newEslError("", ErrCouldNotReadBody, err)
			}
		} else {
			m.Body = nil
		}
	}

	return nil
}

// Dump - Will return message prepared to be dumped out. It's like prettify message for output
func (m *Message) Dump() (resp string) {
	var keys []string

	for k := range m.Headers {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		resp += fmt.Sprintf("%s: %s\r\n", k, m.Headers[k])
	}

	resp += fmt.Sprintf("BODY: %v\r\n", string(m.Body))

	return
}

// newMessage - Will build and execute parsing against received freeswitch message.
// As return will give brand new Message{} for you to use it.
func newMessage(r *bufio.Reader, autoParse bool) (*Message, error) {

	msg := Message{
		r:       r,
		tr:      textproto.NewReader(r),
		Headers: make(map[string]string),
	}

	if autoParse {
		if err := msg.Parse(); err != nil {
			return &msg, err
		}
	}

	return &msg, nil
}
