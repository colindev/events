package main

import (
	"container/list"
	"crypto/sha1"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/listener"
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

func hash(ev event.Event, rd event.RawData) string {

	s := sha1.New()
	s.Write(ev.Bytes())
	s.Write([]byte(":"))
	s.Write(rd.Bytes())
	return fmt.Sprintf("%x", s.Sum(nil))
}

type Line struct {
	hash                  string
	name                  event.Event
	raw                   event.RawData
	timeRedis, timeEvents time.Time
	cntRedis, cntEvents   int
}

func (l *Line) Hash() string {
	if l.hash == "" {
		l.hash = hash(l.name, l.raw)
	}
	return l.hash
}

type Cache struct {
	list *list.List
	sync.RWMutex
	keep int
}

func (c *Cache) Append(from string, ev event.Event, rd event.RawData) *Line {
	c.Lock()
	defer c.Unlock()

	h := hash(ev, rd)

	el := c.list.Back()
	var line *Line

	for {
		if el == nil {
			break
		}
		if l := el.Value.(*Line); l.hash == h {
			line = l
			break
		}
		el = el.Prev()
	}

	if line == nil {
		line = &Line{
			hash: h,
			name: ev,
			raw:  rd,
		}
		c.list.PushBack(line)
		for c.list.Len() > c.keep {
			c.list.Remove(c.list.Front())
		}
	}

	switch from {
	case "redis":
		line.cntRedis++
		line.timeRedis = time.Now()
	case "events":
		line.cntEvents++
		line.timeEvents = time.Now()
	}

	return line
}

var (
	version string
)

func init() {
	log.SetPrefix("[" + version + "]")
}

func main() {

	var (
		verbose    bool
		showVer    bool
		redisAddr  string
		eventsAddr string
		chs        channels
		keep       int
		reconn     = time.Second * 3

		cli = flag.CommandLine
	)

	cli.BoolVar(&verbose, "V", false, "verbose")
	cli.BoolVar(&showVer, "v", false, "version")
	cli.StringVar(&redisAddr, "redis", ":6379", "redis addr")
	cli.StringVar(&eventsAddr, "events", ":6300", "events addr")
	cli.Var(&chs, "event", "watch events")
	cli.IntVar(&keep, "keep", 50, "keep lines")

	cli.Parse(os.Args[1:])

	if verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	if showVer {
		fmt.Println(version)
		os.Exit(0)
	}

	cache := &Cache{
		list: list.New(),
		keep: keep,
	}

	redisListener := redis.NewListener(redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", redisAddr)
	}, 2))

	redisListener.On("*", func(ev event.Event, rd event.RawData) {
		cache.Append("redis", ev, rd)
	})

	eventsListener := listener.New(func() (client.Conn, error) {
		return client.Dial("", eventsAddr)
	})
	eventsListener.On("*", func(ev event.Event, rd event.RawData) {
		cache.Append("events", ev, rd)
	})

	quitRedis := make(chan os.Signal, 1)
	go redisListener.RunForever(quitRedis, reconn, chs...)
	quitEvents := make(chan os.Signal, 1)
	go eventsListener.RunForever(quitEvents, reconn, chs...)

	tk := time.NewTicker(time.Second * 5)
	for t := range tk.C {
		cache.RLock()
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()

		fmt.Println(t)
		el := cache.list.Front()
		var cntMiss, cntDup int
		for {
			if el == nil {
				break
			}

			color := "\033[32m"
			line := el.Value.(*Line)
			if line.cntRedis == 0 || line.cntEvents == 0 {
				color = "\033[31m"
				cntMiss++
			} else if line.cntRedis > 1 || line.cntEvents > 1 {
				color = "\033[35m"
				cntDup++
			}

			fmt.Printf("%s%s redis: %d / events: %d\033[m \033[2m%s\033[m\n", color, line.name, line.cntRedis, line.cntEvents, line.Hash())
			if verbose {
				fmt.Printf("\033[2;33m%s (redis)\033[m\n", line.timeRedis)
				fmt.Printf("\033[2;33m%s (events)\033[m\n", line.timeEvents)
				fmt.Printf("\033[2m%s\033[m\n", line.raw.String())
			}
			el = el.Next()
		}
		cache.RUnlock()
		fmt.Printf("miss: %d/%d duplicate: %d/%d\n", cntMiss, cache.keep, cntDup, cache.keep)
	}
}
