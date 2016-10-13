package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/colindev/events/event"
	"github.com/colindev/events/listener"
	"github.com/colindev/events/redis"
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
		listenAddr   string
		listenEvents channels
		cli          = flag.CommandLine
	)

	cli.StringVar(&listenAddr, "listen", ":6379", "listen event address")
	cli.Var(&listenEvents, "event", "listen events")
	cli.Parse(os.Args[1:])

	li := listener.New(redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", listenAddr)
	}, 10))

	handler := buildHandler(cli.Arg(0), cli.Args()[1:])
	for _, ev := range listenEvents {
		li.On(event.Event(ev.(string)), handler)
	}

	li.Run(listenEvents...)
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
