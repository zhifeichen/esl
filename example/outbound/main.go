package main

import (
	"context"

	"github.com/zhifeichen/esl/v2"
	"github.com/zhifeichen/esl/v2/command"
	"github.com/zhifeichen/esl/v2/command/call"
	"github.com/zhifeichen/log"
)

func main() {
	log.Fatal(esl.ListenAndServe(":8888", handle))
}

func handle(ctx context.Context, conn *esl.Connection, response *esl.RawResponse) {
	conn.EnableEvent(ctx, "ALL")
	conn.FilterEvent(esl.EventListenAll, func(e *esl.Event) {
		log.Infof("got event %s\n", e.GetName())
	})


	log.Infof("Got connection!: %#v\n", response)
	log.Infof("response is ok? %t\n", response.IsOk())
	log.Infof("response reply text: %s\n", response.GetReply())

	eventName := response.GetHeader("Event-Name")
	log.Infof("event name: %s\n", eventName)

	if eventName == "CHANNEL_DATA" {
		caller := response.GetHeader("Channel-Caller-Id-Number")
		callee := response.GetHeader("Channel-Destination-Number")
		uniqueID := response.ChannelUUID()
		sdp := response.GetVariable("switch_r_sdp")
		callID := response.GetVariable("sip_call_id")

		log.Infof("caller: %s\ncallee: %s\nchannelId: %s\nsdp: %s\ncallid: %s\n", caller, callee, uniqueID, sdp, callID)

		_, err := conn.SendCommand(ctx, command.Filter{
			EventHeader: "Unique-Id",
			FilterValue: uniqueID,
		})

		// set call-timeout
		err = conn.Set(ctx, "call_timeout", "30", uniqueID)
		if err != nil {
			log.Error(err)
			return
		}

		err = conn.Export(ctx, "hangup_after_bridge", "true", uniqueID)
		if err != nil {
			log.Error(err)
			return
		}

		calleeContactResp, err := conn.SendCommand(ctx, command.API{
			Command: "sofia_contact",
			Arguments: callee,
		})

		if err != nil {
			log.Error(err)
			return
		}

		conn.SendCommand(ctx, &call.Execute{
			UUID: uniqueID,
			AppName: "bridge",
			AppArgs: string(calleeContactResp.Body),
			Sync: true,
		})
		log.Info("send bridge...")
		conn.SendCommand(ctx, &call.Hangup{
			UUID: uniqueID,
			Cause: "{$originate_disposition}",
		})
	}

	select {
	case <- ctx.Done():
		return
	}
}
