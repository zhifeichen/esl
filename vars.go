// Copyright 2020 chen zhifei
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package esl

import "github.com/zhifeichen/log"

var (

	// ReadBufferSize Size of buffer when we read from connection.
	// 1024 << 6 == 65536
	ReadBufferSize = 1024 << 6

	// AvailableMessageTypes Freeswitch events that we can handle (have logic for it)
	AvailableMessageTypes = []string{"auth/request", "text/disconnect-notice", "text/event-json", "text/event-plain", "api/response", "command/reply"}

	// logger
	logger = log.New(log.NewOptions())
)
