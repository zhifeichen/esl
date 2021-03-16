// Copyright 2020 chen zhifei
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package esl

import (
	"errors"
	"fmt"
)

// errors
var (
	ErrInvalidCommandProvided  = errors.New("invalid command provided. Command cannot contain \\r and/or \\n")
	ErrCouldNotReadMIMEHeaders = errors.New("error while reading MIME headers")
	ErrInvalidContentLength    = errors.New("unable to get size of content-length")
	ErrUnsuccessfulReply       = errors.New("got error while reading from reply command")
	ErrCouldNotReadBody        = errors.New("got error while reading reader body")
	ErrUnsupportedMessageType  = errors.New("unsupported message type")
	ErrCouldNotDecode          = errors.New("could not decode/unescape message")
	ErrCouldNotStartListener   = errors.New("got error while attempting to start listener")
	ErrListenerConnection      = errors.New("listener connection error")
	ErrInvalidServerAddr       = errors.New("please make sure to pass along valid address")
	ErrUnexpectedAuthHeader    = errors.New("expected auth/request content type")
	ErrInvalidPassword         = errors.New("could not authenticate against freeswitch with provided password")
	ErrCouldNotCreateMessage   = errors.New("error while creating new message")
	ErrCouldNotSendEvent       = errors.New("must send at least one event header")
	ErrTimeout                 = errors.New("opration timeout")
	ErrConnClosed              = errors.New("Connection closed")
	ErrResponseChn             = errors.New("no response channels")
	ErrNotImplement            = errors.New("not implement")
)

type eslError struct {
	msg    string
	custom error
	err    error
}

func (e eslError) Error() string {
	if len(e.msg) > 0 {
		return fmt.Sprintf("%s: %s: %s", e.msg, e.custom.Error(), e.err.Error())
	}
	return fmt.Sprintf("%s: %s", e.custom.Error(), e.err.Error())
}

func (e *eslError) Is(target error) bool {
	t, ok := target.(*eslError)
	if !ok {
		return e.custom == target || e.err == target
	}
	return e.custom == t.custom
}

func newEslError(msg string, c, e error) *eslError {
	return &eslError{
		msg,
		c,
		e,
	}
}
