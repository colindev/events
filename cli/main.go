package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

var version string

func init() {
	log.SetPrefix("[" + version + "]")
}

func main() {

	var (
		handler event.Handler

		appName       string
		listenAddr    string
		listenEvents  = channels{}
		launcherEvent string
		recoverSince  int64
		recoverUntil  int64
		showVer       bool
		verbose       bool

		cli = flag.CommandLine
	)

	cli.BoolVar(&verbose, "V", false, "verbose")
	cli.BoolVar(&showVer, "v", false, "version")
	cli.StringVar(&appName, "app", "", "app name")
	cli.StringVar(&listenAddr, "server", "127.0.0.1:6300", "listen event address")
	cli.StringVar(&launcherEvent, "fire", "", "fire event {name}:{data}")
	cli.Var(&listenEvents, "event", "listen events")
	cli.Int64Var(&recoverSince, "since", 0, "request recover since")
	cli.Int64Var(&recoverUntil, "until", 0, "request recover until")
	cli.Parse(os.Args[1:])

	if verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
	if showVer {
		fmt.Println(version)
		os.Exit(0)
	}

	// launcher
	la := launcher.New(client.NewPool(func() (client.Conn, error) {
		return client.Dial("", listenAddr)
	}, 30))
	if ev := strings.SplitN(launcherEvent, ":", 2); len(ev) == 2 {
		err := la.Fire(event.Event(ev[0]), event.RawData(ev[1]))
		if verbose {
			log.Printf("fire: %v error(%v)\n", ev, err)
		}
	}
	defer la.Close()

	// listener
	li := listener.New(func() (client.Conn, error) {
		return client.Dial(appName, listenAddr)
	})
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	li.On(event.Ready, func(ev event.Event, _ event.RawData) {
		// Recover
		log.Printf("recover since=%d until=%d\n", recoverSince, recoverUntil)
		li.Recover(recoverSince, recoverUntil)
	}).On(event.Connecting, func(ev event.Event, _ event.RawData) {
		// Connecting
		log.Println(ev)
	}).On(event.Connected, func(ev event.Event, _ event.RawData) {
		// Connected
		log.Println(ev)
	}).On(event.Disconnected, func(ev event.Event, v event.RawData) {
		// Disconnected
		log.Printf("%s: %s\n", ev, v)
	}).RunForever(quit, time.Second*3, listenEvents...)
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
