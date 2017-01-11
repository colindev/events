package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/store"
)

// Channels for flag.Value
type Channels map[string]bool

// Set from flag
func (chs Channels) Set(s string) error {
	if s == "" {
		return errors.New("empty event name")
	}

	chs[s] = true

	return nil
}

func (chs Channels) String() string {
	return fmt.Sprintf("%v", (map[string]bool)(chs))
}

// Date for flag.Value
type Date time.Time

// Set from flag
func (d *Date) Set(s string) (err error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		*d = Date(time.Unix(i, 0))
		return
	}

	t, err := time.Parse(time.RFC3339, s)
	*d = Date(t)
	return
}

func (d *Date) String() string {
	return time.Time(*d).Format(time.RFC3339)
}

var (
	version string
)

func main() {

	var (
		authDSN, eventDSN string
		sinceDate         = Date(time.Unix(0, 0))
		untilDate         = Date(time.Unix(0, 0))
		channels          = Channels{}
		limit             int
		verbose           bool
		showVer           bool
	)

	cli := flag.CommandLine
	cli.Usage = func() {
		cli.PrintDefaults()
		os.Exit(2)
	}
	cli.StringVar(&authDSN, "auth-dsn", "file::memory:", "auth DSN")
	cli.StringVar(&eventDSN, "event-dsn", "file::memory:", "event DSN")
	cli.Var(&sinceDate, "since", "search since")
	cli.Var(&untilDate, "until", "search until")
	cli.IntVar(&limit, "limit", -1, "limit")
	cli.BoolVar(&verbose, "V", false, "verbose")
	cli.BoolVar(&showVer, "v", false, "version")
	cli.Var(&channels, "event", "event name for search")
	cli.Parse(os.Args[1:])

	if showVer {
		fmt.Println(version)
		os.Exit(0)
	}

	if len(channels) == 0 {
		cli.Usage()
	}

	s, err := store.New(store.Config{
		AuthDSN:      authDSN,
		EventDSN:     eventDSN,
		MaxIdleConns: 1,
		MaxOpenConns: 1,
		Debug:        verbose,
		GCDuration:   "999h", // cli 工具不清資料
	})
	if err != nil {
		log.Fatal(err)
	}

	eventPrefix := []string{}
	if !channels["*"] {
		for ev := range channels {
			eventPrefix = append(eventPrefix, event.Event(ev).Type())
		}
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "\033[32msearch %v\033[m\n", channels)
	}

	s.EachEvents(func(e *store.Event) (end error) {

		defer func() {
			limit--
			if limit == 0 {
				end = io.EOF
			}
		}()

		ev, rd, err := client.ParseEvent([]byte(e.Raw))
		if err != nil {
			log.Println(err)
			return
		}

		rd, err = event.Uncompress(rd)
		if err != nil {
			log.Println(err)
			return
		}

		for ch := range channels {
			if event.Event(ch).Match(ev) {
				if verbose {
					fmt.Fprintf(os.Stderr, "\033[32m(\033[33m%d\033[32m) %s\033[m\n", e.ReceivedAt, time.Unix(e.ReceivedAt, 0).Format(time.RFC3339))
				}
				fmt.Println(ev, rd.String())
				break
			}
		}

		return
	}, eventPrefix, time.Time(sinceDate).Unix(), time.Time(untilDate).Unix())

}
