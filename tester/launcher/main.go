package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/launcher"
)

func main() {

	eventName := event.Event("test.event")

	l := launcher.New(client.NewPool(func() (client.Conn, error) {
		return client.Dial("", ":6300")
	}, 20))

	for {
		now := time.Now()
		rand.Seed(int64(now.Nanosecond()))

		data := event.RawData(fmt.Sprintf("%s", now))
		fmt.Println(eventName, data)
		l.Fire(eventName, data)

		time.Sleep(time.Millisecond * 200 * time.Duration(rand.Intn(10)))
	}
}
