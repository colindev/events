package events

import (
	"sync"
	"testing"
	"time"

	"github.com/colindev/events/event"
	"github.com/colindev/events/launcher"
	"github.com/colindev/events/listener"
	"github.com/colindev/events/redis"
)

func TestListenerAndLauncher(t *testing.T) {
	pool := redis.NewPool(func() (redis.Conn, error) { return redis.Dial("tcp", "127.0.0.1:6379") }, 10)
	wg := &sync.WaitGroup{}

	li := listener.New(pool)
	go li.On(event.Event("test.*"), func(ev event.Event, rd event.RawData) {
		t.Log(ev, rd.String())
		wg.Done()
	}).On(event.PONG, func(ev event.Event, rd event.RawData) {
		t.Log(ev, rd.String())
		wg.Done()
	}).Run("test.*")

	wg.Add(3)
	time.Sleep(time.Millisecond * 50)

	if err := li.Ping("x"); err != nil {
		t.Error(err)
		return
	}

	la := launcher.New(pool)
	la.Fire("test.a", event.RawData("A"))
	la.Fire("test.b", event.RawData("B"))

	wg.Wait()

}
