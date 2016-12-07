package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	eventsLauncher "github.com/colindev/events/launcher"
	eventsListener "github.com/colindev/events/listener"
	"github.com/garyburd/redigo/redis"
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
		flow string
		chs  = channels{}
	)

	flag.StringVar(&flow, "flow", "redis://127.0.0.1:6379|events://127.0.0.1:6300", "event flow")
	flag.Var(&chs, "event", "subscribe events")
	flag.Parse()

	s := strings.SplitN(flow, "|", 2)
	if len(s) != 2 {
		log.Fatal(errFlowSchema)
	}

	notifyer, err := parseNotifyer(s[0])
	if err != nil {
		log.Fatal(err)
	}

	receiver, err := parseReceiver(s[1])
	if err != nil {
		log.Fatal(err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	notifyer.Notify(receiver).Run(shutdown, chs...)

}

var (
	errFlowSchema   = errors.New("flow schema error expect [redis://host:port|events://host:port]")
	errNotifyerAddr = errors.New("notifyer addr error")
)

type Receiver interface {
	Fire(event.Event, event.RawData) error
}

type Notifyer struct {
	from eventsListener.Listener
	to   Receiver
}

func (rds *Notifyer) Run(shutdown <-chan os.Signal, chs ...interface{}) error {

	if rds.from == nil {
		return errors.New("請先指定發送端")
	}
	if rds.to == nil {
		return errors.New("請先指定接收端")
	}

	for _, ch := range chs {
		s := fmt.Sprintf("%v", ch)
		rds.from.On(event.Event(s), func(ev event.Event, v event.RawData) {
			rds.to.Fire(ev, v)
		})
	}

	go rds.from.Run(chs...)

	<-shutdown
	return rds.from.Stop()
}

func (rds *Notifyer) Notify(to Receiver) *Notifyer {
	rds.to = to
	return rds
}

func parseNotifyer(addr string) (*Notifyer, error) {

	s := strings.SplitN(addr, "://", 2)
	if len(s) != 2 {
		return nil, errNotifyerAddr
	}
	addr = s[1]

	var from eventsListener.Listener
	switch s[0] {
	case "redis":
		from = NewRedisListener(redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		}, 2))
	case "events":
		from = eventsListener.New(func() (client.Conn, error) {
			return client.Dial("", addr)
		})
	}

	if from == nil {
		return nil, errors.New("[Listener] 不支援該 server 事件接收")
	}

	return &Notifyer{from: from}, nil
}

func parseReceiver(addr string) (Receiver, error) {
	s := strings.SplitN(addr, "://", 2)
	if len(s) != 2 {
		return nil, errNotifyerAddr
	}
	addr = s[1]

	var to Receiver
	switch s[0] {
	case "redis":
		to = NewRedisLauncher(redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		}, 10))
	case "events":
		to = eventsLauncher.New(client.NewPool(func() (client.Conn, error) {
			return client.Dial("", addr)
		}, 10))
	}

	if to == nil {
		return nil, errors.New("[Launcher] 不支援該 server 事件轉發")
	}

	return to, nil
}
