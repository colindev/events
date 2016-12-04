package events

import (
	"log"
	"sync"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/launcher"
	"github.com/colindev/events/listener"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func Example() {

	wg := &sync.WaitGroup{}

	dial := func() (client.Conn, error) {
		return client.Dial("", "127.0.0.1:8000")
	}

	li := listener.New(dial)
	go li.On(event.Event("test.*"), func(ev event.Event, rd event.RawData) {
		log.Println(ev, rd.String())
		wg.Done()
	}).On(event.PONG, func(ev event.Event, rd event.RawData) {
		log.Println(ev, rd.String())
		wg.Done()
	}).Run("test.*")

	wg.Add(3)
	time.Sleep(time.Millisecond * 50)

	if err := li.Ping("x"); err != nil {
		log.Println(err)
		return
	}

	la := launcher.New(client.NewPool(dial, 10))
	la.Fire("test.a", event.RawData("A"))
	la.Fire("test.b", event.RawData("B"))

	wg.Wait()

}
