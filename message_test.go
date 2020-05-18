// Copyright 2020 chen zhifei
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package esl

import (
	"bufio"
	"strings"
	"testing"
)

var (
	msg1 = `Content-Type: command/reply
Reply-Text: +OK Job-UUID: 7d6594c1-42ba-4260-a4e1-a82a3d63bc4a
Job-UUID: 7d6594c1-42ba-4260-a4e1-a82a3d63bc4a
`
  msg2 = `Content-Length: 908
Content-Type: text/event-plain

Event-Name: BACKGROUND_JOB
Core-UUID: ab97dc06-9369-11ea-8aeb-65fa8de67664
FreeSWITCH-Hostname: localhost.localdomain
FreeSWITCH-Switchname: localhost.localdomain
FreeSWITCH-IPv4: 192.168.135.134
FreeSWITCH-IPv6: %3A%3A1
Event-Date-Local: 2020-05-14%2017%3A35%3A30
Event-Date-GMT: Thu,%2014%20May%202020%2009%3A35%3A30%20GMT
Event-Date-Timestamp: 1589448930194500
Event-Calling-File: mod_event_socket.c
Event-Calling-Function: api_exec
Event-Calling-Line-Number: 1525
Event-Sequence: 34613
Job-UUID: 7d6594c1-42ba-4260-a4e1-a82a3d63bc4a
Job-Command: status
Content-Length: 330

UP 0 years, 3 days, 0 hours, 7 minutes, 45 seconds, 418 milliseconds, 715 microseconds
FreeSWITCH (Version 1.4.20  64bit) is ready
18 session(s) since startup`+`
0 session(s) - peak 2, last 5min 0 `+`
0 session(s) per Sec out of max 30, peak 2, last 5min 0 `+`
1000 session(s) max
min idle cpu 0.00/99.60
Current Stack Size/Max 240K/8192K
`
msg3 = `Content-Type: auth/request
`

msg4 = `Content-Type: command/reply
Reply-Text: +OK accepted
`
msg5 = `Content-Type: command/reply
Reply-Text: +OK event listener enabled json
`
msg6 = `Content-Type: command/reply
Reply-Text: +OK Job-UUID: 7370aac1-93a2-4155-8c4a-2997edd4176b
Job-UUID: 7370aac1-93a2-4155-8c4a-2997edd4176b
`
msg7 = `Content-Length: 949
Content-Type: text/event-json

{"Event-Name":"BACKGROUND_JOB","Core-UUID":"ab97dc06-9369-11ea-8aeb-65fa8de67664","FreeSWITCH-Hostname":"localhost.localdomain","FreeSWITCH-Switchname":"localhost.localdomain","FreeSWITCH-IPv4":"192.168.135.134","FreeSWITCH-IPv6":"::1","Event-Date-Local":"2020-05-11 17:40:25","Event-Date-GMT":"Mon, 11 May 2020 09:40:25 GMT","Event-Date-Timestamp":"1589190025953014","Event-Calling-File":"mod_event_socket.c","Event-Calling-Function":"api_exec","Event-Calling-Line-Number":"1525","Event-Sequence":"651","Job-UUID":"7370aac1-93a2-4155-8c4a-2997edd4176b","Job-Command":"status","Content-Length":"330","_body":"UP 0 years, 0 days, 0 hours, 12 minutes, 41 seconds, 178 milliseconds, 309 microseconds\nFreeSWITCH (Version 1.4.20  64bit) is ready\n0 session(s) since startup\n0 session(s) - peak 0, last 5min 0 \n0 session(s) per Sec out of max 30, peak 0, last 5min 0 \n1000 session(s) max\nmin idle cpu 0.00/99.10\nCurrent Stack Size/Max 240K/8192K\n"}`
)

func Test_newMessage(t *testing.T) {
	type args struct {
		r         *bufio.Reader
		autoParse bool
	}
	tests := []struct {
		name    string
		msg string
		args    args
		wantErr bool
	}{
		{"msg1", msg1, args{bufio.NewReader(strings.NewReader(msg1)), true}, false},
		{"msg2", msg2, args{bufio.NewReader(strings.NewReader(msg2)), true}, false},
		{"msg3", msg3, args{bufio.NewReader(strings.NewReader(msg3)), true}, false},
		{"msg4", msg4, args{bufio.NewReader(strings.NewReader(msg4)), true}, false},
		{"msg5", msg5, args{bufio.NewReader(strings.NewReader(msg5)), true}, false},
		{"msg6", msg6, args{bufio.NewReader(strings.NewReader(msg6)), true}, false},
		{"msg7", msg7, args{bufio.NewReader(strings.NewReader(msg7)), true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newMessage(tt.args.r, tt.args.autoParse)
			if (err != nil) != tt.wantErr {
				t.Errorf("newMessage() error = %v, wantErr %v", err, tt.wantErr)
				t.Errorf("msg: %s", tt.msg)
				return
			}
			if got == nil {
				t.Errorf("newMessage() = %v", got)
			}
		})
	}
}
