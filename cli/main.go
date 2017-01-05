package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
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

// Date of since until
type Date time.Time

func (d *Date) String() string {
	return time.Time(*d).String()
}

// Set implement for flag
func (d *Date) Set(s string) error {
	i, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		*d = Date(time.Unix(i, 0))
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		*d = Date(t)
		return nil
	}

	return fmt.Errorf("must be RFC3339 %s or timestamp", time.RFC3339)
}

var version string

func init() {
	log.SetPrefix("[" + version + "]")
}

func main() {

	var (
		handler event.Handler

		appName       string
		target        string
		listenAddr    string
		listenEvents  = channels{}
		launcherEvent string
		recoverSince  = Date(time.Unix(0, 0))
		recoverUntil  = Date(time.Unix(0, 0))
		interactive   bool
		showInfo      bool
		showVer       bool
		verbose       bool

		cli = flag.CommandLine
	)

	cli.BoolVar(&verbose, "V", false, "verbose")
	cli.BoolVar(&showVer, "v", false, "version")
	cli.BoolVar(&showInfo, "i", false, "show server info")
	cli.BoolVar(&interactive, "I", false, "interactive mode")
	cli.StringVar(&appName, "app", "", "app name")
	cli.StringVar(&target, "to", "", "fire to specify app")
	cli.StringVar(&listenAddr, "server", "127.0.0.1:6300", "listen event address")
	cli.StringVar(&launcherEvent, "fire", "", "fire event {name}:{data}")
	cli.Var(&listenEvents, "event", "listen events")
	cli.Var(&recoverSince, "since", fmt.Sprintf("request recover since, use RFC3339 %s or timestamp", time.RFC3339))
	cli.Var(&recoverUntil, "until", fmt.Sprintf("request recover until, use RFC3339 %s or timestamp", time.RFC3339))
	cli.Parse(os.Args[1:])

	if verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
	if showVer {
		fmt.Println(version)
		os.Exit(0)
	}

	var (
		la launcher.Launcher
		li = listener.New(func() (client.Conn, error) {
			return client.Dial(appName, listenAddr)
		})
	)

	if launcherEvent != "" || interactive {
		la = launcher.New(client.NewPool(func() (client.Conn, error) {
			return client.Dial("", listenAddr)
		}, 30))
	}

	if interactive {
		go func() {
			stdin := bufio.NewReader(os.Stdin)
			for {
				line, _, err := stdin.ReadLine()
				if err != nil {
					fmt.Println(err)
					continue
				}

				if len(line) == 0 {
					continue
				}

				i := bytes.IndexByte(line, ' ')
				var cmd string
				if i == -1 {
					cmd = string(line)
				} else {
					cmd = string(line[:i])
				}
				switch strings.ToUpper(cmd) {
				case "FIRE":
					ev, rd, err := client.ParseEvent(line[i+1:])
					if err != nil {
						fmt.Println(err)
						continue
					}
					la.Fire(ev, rd)
				case "FIRETO":
					var (
						target string
						b      []byte
					)
					_, err := fmt.Sscan(string(line[i+1:]), &target, &b)
					if err != nil {
						fmt.Println(err)
						continue
					}
					ev, rd, err := client.ParseEvent(b)
					if err != nil {
						fmt.Println(err)
						continue
					}
					la.FireTo(target, ev, rd)
				case "INFO":
					li.Info()
				default:
					fmt.Println("unkown command")
				}

			}
		}()
	}

	// launcher
	if ev := strings.SplitN(launcherEvent, ":", 2); len(ev) == 2 {
		var err error
		if target == "" {
			err = la.Fire(event.Event(ev[0]), event.RawData(ev[1]))
		} else {
			err = la.FireTo(target, event.Event(ev[0]), event.RawData(ev[1]))
		}
		if verbose {
			log.Printf("fire: %v error(%v)\n", ev, err)
		}
		defer la.Close()
	}

	// listener
	quit := make(chan os.Signal, 1)
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
		if !showInfo {
			return
		}
		// display and close
		li.On(event.Info, func(ev event.Event, rd event.RawData) {
			handler(ev, rd)
			quit <- syscall.SIGQUIT
		})
	} else {
		li.On(event.Info, func(ev event.Event, rd event.RawData) {
			handler(ev, rd)
		})
	}

	for _, ev := range listenEvents {
		li.On(event.Event(ev.(string)), handler)
	}

	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	li.On(event.Ready, func(ev event.Event, _ event.RawData) {
		// Recover
		since := time.Time(recoverSince).Unix()
		until := time.Time(recoverUntil).Unix()
		log.Printf("recover since=%d until=%d\n", since, until)
		li.Recover(since, until)
	}).On(event.Connecting, func(ev event.Event, _ event.RawData) {
		// Connecting
		log.Println(ev)
	}).On(event.Connected, func(ev event.Event, _ event.RawData) {
		// Connected
		log.Println(ev)

		if showInfo {
			li.Info()
		}
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
