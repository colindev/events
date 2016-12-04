package main

import (
	"log"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/listener"
)

func main() {

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	l := listener.New(func() (client.Conn, error) {
		return client.Dial("tester", "127.0.0.1:8000")
	})

	l.On(event.Event("*"), func(ev event.Event, rd event.RawData) {
		log.Println(ev, rd.String())
	})

	// 斷線自動重連
	for {
		err := l.Run("a", "b", "c")
		if err != nil {
			log.Println(err)
		}

		time.Sleep(time.Second * 3)
	}

}
