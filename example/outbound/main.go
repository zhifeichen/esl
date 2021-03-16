package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhifeichen/esl/v2"
	"github.com/zhifeichen/esl/v2/command"
	"github.com/zhifeichen/esl/v2/command/call"
	"github.com/zhifeichen/log"
)

const (
	channelHangup          = "CHANNEL_HANGUP"
	channelHangupComplete  = "CHANNEL_HANGUP_COMPLETE"
	channelExecuteComplete = "CHANNEL_EXECUTE_COMPLETE"
)

func main() {
	go func() {
		log.Fatal(esl.ListenAndServe(":8888", handle))
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutdown esl outbound server...")
	esl.Shutdown()
}

func handle(ctx context.Context, conn *esl.Connection) {
	response, err := conn.SendCommand(ctx, command.Connect{})
	if err != nil {
		log.Errorf("Error connecting to %s error %s\n", conn.RemoteAddr().String(), err.Error())
		conn.Close()
		return
	}

	conn.EnableEvent(ctx, "ALL")
	// eventChn := make(chan *esl.Event, 10)
	// go eventHandle(ctx, conn, eventChn)
	conn.FilterEvent(esl.EventListenAll, func(event *esl.Event) {
		log.Infof("got event %s\n", event.GetName())
		// eventChn <- e
		go eventHandle(ctx, conn, event)
	})

	_, err = conn.SendCommand(ctx, command.Linger{
		Enabled:  true,
		Duration: 10,
	})
	if err != nil {
		log.Error(err)
		return
	}

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

		conn.SendCommand(ctx, command.Filter{
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
			Command:   "sofia_contact",
			Arguments: callee,
		})

		if err != nil {
			log.Error(err)
			return
		}

		conn.SendCommand(ctx, &call.Execute{
			UUID:    uniqueID,
			AppName: "bridge",
			AppArgs: string(calleeContactResp.Body),
			Sync:    true,
		})
		log.Info("send bridge...")
	}

	<-ctx.Done()
	log.Info("handle ctx done")
}

func eventHandle(ctx context.Context, conn *esl.Connection, event *esl.Event) {

	name := event.GetName()
	application := event.GetHeader("Application")
	log.Infof("got event[%s], application: %s\n", name, application)
	cmd, err := doEvent(event)
	if err != nil {
		log.Error(err)
		return
	}
	if cmd != nil {
		resp, err := conn.SendCommand(context.TODO(), call.Hangup{
			UUID:  event.ChannelUUID(),
			Cause: event.GetVariable("originate_disposition"),
		})
		if err != nil {
			log.Error(err)
			return
		}
		log.Infof("hungup response: %#v\n", resp)
	}

}

func doEvent(event *esl.Event) (command.Command, error) {
	name := event.GetName()
	switch name {
	case channelExecuteComplete:
		app := event.GetHeader("Application")
		if app == "bridge" {
			return doBridgeComplete(event)
		}
	case channelHangupComplete:
		return doHangupComplete(event)
	}
	return nil, nil
}

func doBridgeComplete(event *esl.Event) (command.Command, error) {
	uuid := event.ChannelUUID()
	cause := event.GetVariable("originate_disposition")
	if cause != "SUCCESS" {
		return call.Hangup{
			UUID:  uuid,
			Cause: cause,
		}, nil
	}
	return nil, nil
}

func doHangupComplete(event *esl.Event) (command.Command, error) {
	log.Infof("hangup complete:\nduration: %s\nbillsec: %s", event.GetVariable("duration"), event.GetVariable("billsec"))
	return nil, nil
}
