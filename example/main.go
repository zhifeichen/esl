package main

import (
	"encoding/json"
	"flag"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/google/uuid"
	"github.com/zhifeichen/log"
)

var (
	fshost   = flag.String("fshost", "192.168.135.134", "Freeswitch hostname. Default: localhost")
	fsport   = flag.Uint("fsport", 8021, "Freeswitch port. Default: 8021")
	password = flag.String("pass", "ClueCon", "Freeswitch password. Default: ClueCon")
	timeout  = flag.Int("timeout", 10, "Freeswitch conneciton timeout in seconds. Default: 10")
)

// Small client that will first make sure all events are returned as JSON and second, will originate
func main() {
	flag.Parse()
	rl, err := readline.NewEx(&readline.Config{
		// UniqueEditLine: true,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	// log.SetOutput(rl.Stderr())
	log.Init(log.NewOptions(
		log.Level("trace"),
		log.Writers(rl.Stdout()),
	))

	// Boost it as much as it can go ...
	runtime.GOMAXPROCS(runtime.NumCPU())

	client := eslclient{
		host: *fshost,
		port: uint16(*fsport),
		auth: *password,
	}

	// Apparently all is good... Let us now handle connection :)
	// We don't want this to be inside of new connection as who knows where it my lead us.
	// Remember that this is crutial part in handling incoming messages :)
	err = client.start()
	if err != nil {
		panic(err)
	}

	log.Debugf("%#v", client)

	client.client.FilterEvent("CDR", func(en string, h map[string]string, body string) {
		log.Debugf("got event[%s]\n", en)
		log.Debug("header: ", h)
		log.Debug("body: ", body)
	})

	client.client.FilterEvent("CHANNEL_HANGUP_COMPLETE", func(en string, h map[string]string, body string) {
		j, _ := json.Marshal(h)

		log.Infof("got event[%s]\n", en)
		log.Info("header: ", string(j))
		log.Info("body: ", body)
	})

	rl.ResetHistory()
	rl.SetPrompt("bgapi>")
	for {
		ln := rl.Line()
		if ln.CanContinue() {
			continue
		} else if ln.CanBreak() {
			break
		}
		cmd := ln.Line
		if len(cmd) == 0 {
			continue
		}
		if cmd == "bye" {
			break
		}
		if strings.Compare(cmd[:4], "api ") == 0 {
			var api []string
			if strings.Contains(cmd, `"`) {
				api = strings.Split(cmd, `"`)
			} else if strings.Contains(cmd, `'`) {
				api = strings.Split(cmd, `'`)
			} else {
				api = strings.Split(cmd, " ")
			}
			// api := strings.Split(cmd, " ")
			if len(api) < 2 {
				continue
			}
			log.Debugf("%q\n", api)
			duration := time.Second
			if len(api) > 2 {
				dstr := strings.TrimSpace(api[2])
				dint, err := strconv.Atoi(dstr)
				log.Debugf("dstr: %s, dint: %d\n", dstr, dint)
				if err != nil {
					duration = time.Duration(dint) * time.Second
				}
			}
			h, b, err := client.client.APIWithTimeout(api[1], duration)
			if err != nil {
				log.Error(err)
				continue
			}
			log.Info("header: ", h)
			log.Info("body: ", b)
			continue
		}
		jobuuid := uuid.New().String()
		client.client.BgAPI(cmd, jobuuid, func(h map[string]string, body string) {
			log.Info("header: ", h)
			log.Info("body: ", body)
		})
	}
	rl.Clean()
}
