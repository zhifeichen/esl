package main

import (
	"strings"

	"github.com/zhifeichen/esl/v2"
	"github.com/zhifeichen/log"
)

type eslclient struct {
	host    string
	port    uint16
	auth    string
	running bool
	client  *esl.Client
}

// param type struct
type recordParam struct {
	AccessCode string `json:"accessCode"`
	Subject    string `json:"subject"`
	Type1      int    `json:"type1"`
	Order1     string `json:"order1"`
	Type2      int    `json:"type2"`
	Order2     string `json:"order2"`
	Type3      int    `json:"type3"`
	Order3     string `json:"order3"`
}

var (
	// events = []string{"BACKGROUND_JOB", "CHANNEL_CALLSTATE", "CHANNEL_CREATE", "CUSTOM", "conference::maintenance"}
	events = []string{"BACKGROUND_JOB", "CHANNEL_CALLSTATE", "CHANNEL_CREATE", "CHANNEL_HANGUP_COMPLETE", "CDR", "CUSTOM", "conference::maintenance", "sofia::register", "sofia::unregister"}
	// filters = []struct{h, v string; cb esl.headerFilterCallback}{
	// 	{"Answer-State", "ringing", nil},
	// 	{"Answer-State", "answered", nil},
	// 	{"Answer-State", "hangup", nil},
	// 	{"Action", "conference-create", nil},
	// 	{"Action", "play-file-done", nil},
	// 	{"Action", "conference-destroy", nil},
	// 	{"Action", "add-member", nil},
	// 	{"Action", "del-member", nil},
	// 	{"Action", "bgdial-result", nil},
	// 	{"Action", "audio-ssrc", nil},
	// 	{"Action", "video-ssrc", nil},
	// }
	headerFilters = []struct {
		h, v string
		cb   esl.EventHandler
	}{
		{"Event-Subclass", "sofia::register", func(e *esl.Event) {
			log.Infof("got Event-Subclass: %s\n", e.GetHeader("Event-Subclass"))
			log.Infof("event: %#v\n", e.String())
		}},
		{"Event-Subclass", "sofia::unregister", func(e *esl.Event) {
			log.Infof("got Event-Subclass: %s\n", e.GetHeader("Event-Subclass"))
			log.Infof("event: %#v\n", e.String())
		}},
	}
)

func (c *eslclient) start() error {
	client, err := esl.NewClient(c.host, c.port, c.auth, 3)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("%#v\n", client)
	c.client = client
	err = c.client.Start("plain", strings.Join(events, " "))
	if err != nil {
		return err
	}
	for _, f := range headerFilters {
		c.client.FilterHeader(f.h, f.v, f.cb)
	}
	return nil
}

func (c *eslclient) stop() {
	c.client.Stop()
}
