package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/launcher"
	"github.com/colindev/events/listener"
)

type channels []interface{}

func (chs *channels) String() string {
	return fmt.Sprintf("%v", *chs)
}
func (chs *channels) Set(ev string) error {
	*chs = append(*chs, ev)
	return nil
}

func main() {

	var (
		handler event.Handler

		appName       string
		listenAddr    string
		listenEvents  = channels{}
		launcherEvent string
		doRecover     bool

		cli = flag.CommandLine
	)

	cli.StringVar(&appName, "app", "", "app name")
	cli.StringVar(&listenAddr, "listen", "127.0.0.1:8000", "listen event address")
	cli.StringVar(&launcherEvent, "fire", "", "fire event {name}:{data}")
	cli.Var(&listenEvents, "event", "listen events")
	cli.BoolVar(&doRecover, "r", false, "request recover")
	cli.Parse(os.Args[1:])

	dial := func() (client.Conn, error) {
		return client.Dial(appName, listenAddr)
	}

	// launcher
	la := launcher.New(client.NewPool(dial, 30))
	if ev := strings.SplitN(launcherEvent, ":", 2); len(ev) == 2 {
		log.Println(la.Fire(event.Event(ev[0]), event.RawData(ev[1])))
	}

	// listener
	li := listener.New(dial)
	args := cli.Args()
	switch len(args) {
	case 0:
		handler = buildHandler("echo", []string{})
	case 1:
		handler = buildHandler(args[0], []string{})
	default:
		handler = buildHandler(args[0], args[1:])
	}

	if len(listenEvents) == 0 {
		return
	}

	for _, ev := range listenEvents {
		li.On(event.Event(ev.(string)), handler)
	}

	li.On(listener.Connected, func(ev event.Event, _ event.RawData) {
		li.Recover()
	}).Run(listenEvents...)
}

func buildHandler(cmdName string, cmdArgs []string) event.Handler {
	return func(ev event.Event, rd event.RawData) {
		cmd := exec.Command(cmdName, append(cmdArgs, string(ev), string(rd))...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Println(err)
		}
	}
}
