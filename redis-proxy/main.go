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
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	eventsLauncher "github.com/colindev/events/launcher"
	eventsListener "github.com/colindev/events/listener"
	"github.com/colindev/events/redis-proxy/redis"
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
		flow    string
		chs     = channels{}
		showVer bool
		verbose bool
	)

	flag.BoolVar(&verbose, "V", false, "verbose")
	flag.BoolVar(&showVer, "v", false, "version")
	flag.StringVar(&flow, "flow", "redis://127.0.0.1:6379|events://127.0.0.1:6300", "event flow")
	flag.Var(&chs, "event", "subscribe events")
	flag.Parse()

	if verbose {
		log.SetFlags(log.Lshortfile | log.LstdFlags)
	}

	if showVer {
		fmt.Println("redis-proxy: ", version)
		os.Exit(0)
	}

	s := strings.SplitN(flow, "|", 2)
	if len(s) != 2 {
		log.Fatal(errFlowSchema)
	}

	notifyer, err := parseNotifyer(s[0])
	if err != nil {
		log.Fatal(err)
	}

	receiver, mode, err := parseReceiver(s[1])
	if err != nil {
		log.Fatal(err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	notifyer.Notify(receiver, mode).Run(shutdown, chs...)

}

var (
	errFlowSchema   = errors.New("flow schema error expect [redis://host:port|events://host:port]")
	errNotifyerAddr = errors.New("notifyer addr error")
)

type Receiver interface {
	Fire(event.Event, event.RawData) error
	Close() error
}

type Notifyer struct {
	fromType string
	from     eventsListener.Listener
	toType   string
	to       Receiver
}

func (rds *Notifyer) Run(shutdown chan os.Signal, chs ...interface{}) error {

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

	log.Printf("run: \033[32m%s\033[m -> \033[32m%s\033[m\n", rds.fromType, rds.toType)
	rds.from.RunForever(shutdown, time.Second*3, chs...)
	log.Printf("try stop listener(%s)\n", rds.fromType)
	rds.from.WaitHandler()
	log.Printf("try stop launcher(%s)\n", rds.toType)
	return rds.to.Close()
}

func (rds *Notifyer) Notify(to Receiver, toType string) *Notifyer {
	rds.to = to
	rds.toType = toType
	return rds
}

func parseNotifyer(addr string) (*Notifyer, error) {

	s := strings.SplitN(addr, "://", 2)
	if len(s) != 2 {
		return nil, errNotifyerAddr
	}
	addr = s[1]

	var (
		from eventsListener.Listener
	)
	switch s[0] {
	case "redis":
		from = redis.NewListener(redis.NewPool(func() (redis.Conn, error) {
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

	return &Notifyer{
		from:     from,
		fromType: s[0],
	}, nil
}

func parseReceiver(addr string) (Receiver, string, error) {
	s := strings.SplitN(addr, "://", 2)
	if len(s) != 2 {
		return nil, "", errNotifyerAddr
	}
	addr = s[1]

	var to Receiver
	switch s[0] {
	case "redis":
		to = redis.NewLauncher(redis.NewPool(func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		}, 10))
	case "events":
		to = eventsLauncher.New(client.NewPool(func() (client.Conn, error) {
			return client.Dial("", addr)
		}, 10))
	}

	if to == nil {
		return nil, s[0], errors.New("[Launcher] 不支援該 server 事件轉發")
	}

	return to, s[0], nil
}
