// Copyright 2020 chen zhifei
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package esl

import (
	"context"
	"net/textproto"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zhifeichen/esl/v2/command"
	"github.com/zhifeichen/log"
)

var (
	testCnt = 200
	rCnt    = 200
	cCnt    = 32
)

func TestConnection_SendCommand(t *testing.T) {
	// type args struct {
	// 	ctx context.Context
	// 	cmd command.Command
	// 	fn  []EventHandler
	// }
	// tests := []struct {
	// 	name    string
	// 	c       *Connection
	// 	args    args
	// 	want    *RawResponse
	// 	wantErr bool
	// }{
	// 	// TODO: Add test cases.
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		got, err := tt.c.SendCommand(tt.args.ctx, tt.args.cmd, tt.args.fn...)
	// 		if (err != nil) != tt.wantErr {
	// 			t.Errorf("Connection.SendCommand() error = %v, wantErr %v", err, tt.wantErr)
	// 			return
	// 		}
	// 		if !reflect.DeepEqual(got, tt.want) {
	// 			t.Errorf("Connection.SendCommand() = %v, want %v", got, tt.want)
	// 		}
	// 	})
	// }

	t1 := time.Now()
	defer func() {
		esc := time.Since(t1).Milliseconds()
		t.Errorf("escap %d ms\n", esc)
	}()

	log.Init(log.NewOptions(
		log.Level("error"),
		log.Writers(os.Stdout),
	))
	host := "192.168.135.134"
	// host := "172.21.24.80"
	port := uint16(18021)
	// port := uint16(8021)
	auth := "ClueCon"
	// clients := make([]*Client, cCnt)
	// var err error
	// for _, c := range clients {
	// 	c, err = NewClient(host, port, auth, 3)
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// 	err = c.Start("plain", "BACKGROUND_JOB")
	// 	if err != nil {
	// 		t.Error(err)
	// 	}
	// 	go c.sendLoop()
	// }
	client, err := NewClient(host, port, auth, 3, cCnt)
	if err != nil {
		t.Error(err)
	}
	err = client.Start("plain", "BACKGROUND_JOB")
	if err != nil {
		t.Error(err)
	}
	// go client.sendLoop()

	cmd := command.SendEvent{
		Name:    "SEND_MESSAGE",
		Headers: make(textproto.MIMEHeader),
		Body:    "test",
	}
	cmd.Headers.Add("User", "19900001111202104fu")
	cmd.Headers.Add("Host", host)
	cmd.Headers.Add("profile", "internal")
	t2 := time.Now()
	// _, err := client.SendCommand(context.Background(), &cmd)
	// if err != nil {
	// 	t.Error(err)
	// }
	wt := sync.WaitGroup{}
	wt.Add(1)
	client.SendCommand2(context.Background(), &cmd, func(e *Event) {
		esc2 := time.Since(t2).Milliseconds()
		t.Errorf("esc: %d\n", esc2)
		wt.Done()
	})
	wt.Wait()
	// return

	// client2, err := NewClient(host, port, auth, 3)
	// if err != nil {
	// 	t.Error(err)
	// }
	// err = client2.Start("plain", "BACKGROUND_JOB")
	// if err != nil {
	// 	t.Error(err)
	// }
	// go client2.sendLoop()

	// client3, err := NewClient(host, port, auth, 3)
	// if err != nil {
	// 	t.Error(err)
	// }
	// err = client3.Start("plain", "BACKGROUND_JOB")
	// if err != nil {
	// 	t.Error(err)
	// }
	// go client3.sendLoop()

	// cmd := command.SendEvent{
	// 	Name:    "SEND_MESSAGE",
	// 	Headers: make(textproto.MIMEHeader),
	// 	Body:    "test",
	// }
	// cmd.Headers.Add("User", "19900001111202104fu")
	// cmd.Headers.Add("Host", host)
	// cmd.Headers.Add("profile", "internal")
	var tt, max, cnt int64 = 0, 0, 0
	wg := sync.WaitGroup{}
	wg.Add(rCnt * testCnt)
	t3 := time.Now()
	for j := 0; j < rCnt; j++ {
		go func() {
			for i := 0; i < testCnt; i++ {
				cmd := command.SendEvent{
					Name:    "SEND_MESSAGE",
					Headers: make(textproto.MIMEHeader),
					Body:    "test",
				}
				cmd.Headers.Add("User", "19900001111202104fu")
				cmd.Headers.Add("Host", host)
				cmd.Headers.Add("profile", "internal")
				t2 := time.Now()
				// _, err := client.SendCommand(context.Background(), &cmd)
				// if err != nil {
				// 	t.Error(err)
				// }
				client.SendCommand2(context.Background(), &cmd, func(e *Event) {
					esc2 := time.Since(t2).Milliseconds()
					// tt += esc2
					atomic.AddInt64(&tt, esc2)
					if esc2 > max {
						max = esc2
					}
					// cnt++
					atomic.AddInt64(&cnt, 1)
					wg.Done()
				})
				// esc2 := time.Since(t2).Microseconds()
				// tt += esc2
				// if esc2 > max {
				// 	max = esc2
				// }
			}
		}()
	}
	wg.Wait()
	t.Errorf("tt: %d; evg esc: %d; max: %d, cnt: %d, chn len: %d, esc: %d\n", tt, tt/int64(cnt), max, cnt, len(client.sendParamChn), time.Since(t3).Milliseconds())
}

func TestConnection_SendCommand2(t *testing.T) {
	t1 := time.Now()
	defer func() {
		esc := time.Since(t1).Milliseconds()
		t.Errorf("escap %d ms\n", esc)
	}()

	log.Init(log.NewOptions(
		log.Level("error"),
		// log.Writers(rl.Stdout()),
	))
	// host := "192.168.135.134"
	host := "172.21.24.80"
	// port := uint16(18021)
	port := uint16(8021)
	auth := "ClueCon"
	client, err := NewClient(host, port, auth, 3, cCnt)
	if err != nil {
		t.Error(err)
	}
	err = client.Start("plain", "BACKGROUND_JOB")
	if err != nil {
		t.Error(err)
	}
	cmd := command.SendEvent{
		Name:    "SEND_MESSAGE",
		Headers: make(textproto.MIMEHeader),
		Body:    "test",
	}
	cmd.Headers.Add("User", "19900001111202104fu")
	cmd.Headers.Add("Host", host)
	cmd.Headers.Add("profile", "internal")
	var tt, max, cnt int64 = 0, 0, 0
	wg := sync.WaitGroup{}
	wg.Add(testCnt * rCnt)
	for i := 0; i < testCnt*rCnt; i++ {
		t2 := time.Now()
		_, err := client.SendCommand(context.Background(), &cmd)
		if err != nil {
			t.Error(err)
		}
		esc2 := time.Since(t2).Milliseconds()
		tt += esc2
		if esc2 > max {
			max = esc2
		}
		cnt++
		wg.Done()
	}
	wg.Wait()
	t.Errorf("tt: %d; evg esc: %d; max: %d, cnt: %d\n", tt, tt/int64(cnt), max, cnt)
}

func TestConnection_SendCommand3(t *testing.T) {
	t1 := time.Now()
	defer func() {
		esc := time.Since(t1).Milliseconds()
		t.Errorf("escap %d ms\n", esc)
	}()

	log.Init(log.NewOptions(
		log.Level("error"),
		// log.Writers(rl.Stdout()),
	))
	// host := "192.168.135.134"
	// host := "172.21.24.117"
	host := "172.21.24.80"
	// port := uint16(18021)
	port := uint16(8021)
	auth := "ClueCon"
	client, err := NewClient(host, port, auth, 3, cCnt)
	if err != nil {
		t.Error(err)
	}
	err = client.Start("plain", "BACKGROUND_JOB")
	if err != nil {
		t.Error(err)
	}
	// cmd := command.SendEvent{
	// 	Name:    "SEND_MESSAGE",
	// 	Headers: make(textproto.MIMEHeader),
	// 	Body:    "test",
	// }
	// cmd.Headers.Add("User", "19900001111202104fu")
	// cmd.Headers.Add("Host", host)
	// cmd.Headers.Add("profile", "internal")
	var tt, max, cnt int64 = 0, 0, 0
	wg := sync.WaitGroup{}
	wg.Add(rCnt * testCnt)
	for j := 0; j < rCnt; j++ {
		go func() {
			for i := 0; i < testCnt; i++ {
				cmd := command.SendEvent{
					Name:    "SEND_MESSAGE",
					Headers: make(textproto.MIMEHeader),
					Body:    "test",
				}
				cmd.Headers.Add("User", "19900001111202104fu")
				cmd.Headers.Add("Host", host)
				cmd.Headers.Add("profile", "internal")
				t2 := time.Now()
				_, err := client.SendCommand(context.Background(), &cmd)
				if err != nil {
					t.Error(err)
				}
				esc2 := time.Since(t2).Milliseconds()
				tt += esc2
				if esc2 > max {
					max = esc2
				}
				cnt++

				wg.Done()
			}
		}()
	}
	wg.Wait()
	t.Errorf("tt: %d; evg esc: %d; max: %d, cnt: %d\n", tt, tt/int64(cnt), max, cnt)
}

func TestConnection_SendCommand4(t *testing.T) {
	t1 := time.Now()
	defer func() {
		esc := time.Since(t1).Milliseconds()
		t.Errorf("escap %d ms\n", esc)
	}()

	log.Init(log.NewOptions(
		log.Level("error"),
		// log.Writers(rl.Stdout()),
	))
	host := "192.168.135.134"
	// host := "172.21.24.117"
	// host := "172.21.24.80"
	port := uint16(18021)
	// port := uint16(8021)
	auth := "ClueCon"
	client, err := NewClient(host, port, auth, 3, 1)
	if err != nil {
		t.Error(err)
	}
	err = client.Start("plain", "BACKGROUND_JOB")
	if err != nil {
		t.Error(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	cmd := command.SendEvent{
		Name:    "SEND_MESSAGE",
		Headers: make(textproto.MIMEHeader),
		Body:    "test",
	}
	cmd.Headers.Add("User", "19900001111202104fu")
	cmd.Headers.Add("Host", host)
	cmd.Headers.Add("profile", "internal")
	client.SendCommand2(context.Background(), &cmd, func(e *Event) {
		t.Errorf("%#v\n", e)
		wg.Done()
	})

	cmd2 := command.API{
		Command:    "status",
		Arguments:  "",
		Background: true,
	}
	client.SendCommand2(context.Background(), cmd2, func(e *Event) {
		t.Errorf("%#v\n", e)
		wg.Done()
	})
	wg.Wait()
}

func Benchmark_SendCommand(b *testing.B) {
	log.Init(log.NewOptions(
		log.Level("fatal"),
		// log.Writers(rl.Stdout()),
	))
	// host := "192.168.135.134"
	host := "172.21.24.80"
	// port := uint16(18021)
	port := uint16(8021)
	auth := "ClueCon"
	client, err := NewClient(host, port, auth, 3, cCnt)
	if err != nil {
		b.Error(err)
	}
	err = client.Start("plain", "BACKGROUND_JOB")
	if err != nil {
		b.Error(err)
	}
	cmd := command.SendEvent{
		Name:    "SEND_MESSAGE",
		Headers: make(textproto.MIMEHeader),
		Body:    "test",
	}
	for i := 0; i < b.N; i++ {
		_, err := client.SendCommand(context.Background(), &cmd)
		if err != nil {
			b.Error(err)
		}
	}
}

func Benchmark_SendCommand2(b *testing.B) {
	log.Init(log.NewOptions(
		log.Level("fatal"),
		// log.Writers(rl.Stdout()),
	))
	host := "192.168.135.134"
	// host := "172.21.24.80"
	port := uint16(18021)
	// port := uint16(8021)
	auth := "ClueCon"
	client, err := NewClient(host, port, auth, 3, cCnt)
	if err != nil {
		b.Error(err)
	}
	err = client.Start("plain", "BACKGROUND_JOB")
	if err != nil {
		b.Error(err)
	}
	cmd := command.SendEvent{
		Name:    "SEND_MESSAGE",
		Headers: make(textproto.MIMEHeader),
		Body:    "test",
	}
	for i := 0; i < b.N; i++ {
		_, err := client.SendCommand(context.Background(), &cmd)
		if err != nil {
			b.Error(err)
		}
	}
}
