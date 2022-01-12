package esl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// EventHandler event handler callback
type EventHandler func(e *Event)

// EventListenAll listen all event key
const EventListenAll = "ALL"

// Event struct
type Event struct {
	Headers textproto.MIMEHeader
	Body    []byte
}

// BgCallback bgapi callback func
type BgCallback func(header map[string]string, body string)

// EventCallback event file callback func
type EventCallback func(eventName string, header map[string]string, body string)

// HeaderFilterCallback filter by header field callback func
type HeaderFilterCallback func(name, value string, header map[string]string, body string)

type bgFilter struct {
	sync.Mutex
	cb map[string]EventHandler
}

type eventFilter struct {
	sync.RWMutex
	cb map[string]EventHandler
}

type headerFilterItem struct {
	header string
	value  string
	cb     EventHandler
}

type headerFilter struct {
	sync.RWMutex
	cb []*headerFilterItem
}

type filter struct {
	bgapi  bgFilter
	event  eventFilter
	header headerFilter
}

func readPlainEvent(body []byte) (*Event, error) {
	reader := bufio.NewReader(bytes.NewBuffer(body))
	header := textproto.NewReader(reader)

	headers, err := header.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	event := &Event{
		Headers: headers,
	}

	if contentLength := headers.Get("Content-Length"); len(contentLength) > 0 {
		length, err := strconv.Atoi(contentLength)
		if err != nil {
			return event, err
		}
		event.Body = make([]byte, length)
		_, err = io.ReadFull(reader, event.Body)
		if err != nil {
			return event, err
		}
	}

	return event, nil
}

// TODO: Needs processing
func readXMLEvent(body []byte) (*Event, error) {
	return nil, ErrNotImplement
}

// TODO: Needs processing
func readJSONEvent(body []byte) (*Event, error) {
	// OK, what is missing here is a way to interpret other JSON types - it expects string only (understandably
		// because the FS events are generally "string: string") - extract into empty interface and migrate only strings.
		// i.e. Event CHANNEL_EXECUTE_COMPLETE - "variable_DP_MATCH":["a=rtpmap:101 telephone-event/8000","101"]
		var decoded map[string]interface{}

		if err := json.Unmarshal(body, &decoded); err != nil {
			return nil, err
		}

		event := &Event{
			Headers: textproto.MIMEHeader{},
		}

		// Copy back in:
		for k, v := range decoded {
			switch v := v.(type) {
			case string:
				event.Headers[textproto.CanonicalMIMEHeaderKey(k)] = append(event.Headers[textproto.CanonicalMIMEHeaderKey(k)], v)
			case []string:
				event.Headers[textproto.CanonicalMIMEHeaderKey(k)] = append(event.Headers[textproto.CanonicalMIMEHeaderKey(k)], v...)
			default:
				//delete(m.Headers, k)
				logger.Warnf("Removed non-string property (%s)", k)
			}
		}

		if v := event.Headers["_body"]; len(v) > 0 && v[0] != "" {
			event.Body = []byte(v[0])
			delete(event.Headers, "_body")
		} else {
			event.Body = []byte("")
		}
		return event, nil
}

// GetName Helper function that returns the event name header
func (e Event) GetName() string {
	return e.GetHeader("Event-Name")
}

// HasHeader Helper to check if the Event has a header
func (e Event) HasHeader(header string) bool {
	_, ok := e.Headers[textproto.CanonicalMIMEHeaderKey(header)]
	return ok
}

// GetHeader Helper function that calls e.Header.Get
func (e Event) GetHeader(header string) string {
	value, _ := url.PathUnescape(e.Headers.Get(header))
	return value
}

// ChannelUUID Helper to get the channel UUID. Calls GetHeader internally
func (e Event) ChannelUUID() string {
	return e.GetHeader("Unique-ID")
}

// GetVariable Helper function to get "Variable_" headers. Calls GetHeader internally
func (e Event) GetVariable(variable string) string {
	return e.GetHeader(fmt.Sprintf("Variable_%s", variable))
}

// String Implement the Stringer interface for pretty printing (%v)
func (e Event) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s\n", e.GetName()))
	for key, values := range e.Headers {
		builder.WriteString(fmt.Sprintf("%s: %#v\n", key, values))
	}
	builder.Write(e.Body)
	return builder.String()
}

// GoString Implement the GoStringer interface for pretty printing (%#v)
func (e Event) GoString() string {
	return e.String()
}
