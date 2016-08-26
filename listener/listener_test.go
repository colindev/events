package listener

import (
	"sync"
	"testing"

	"github.com/colindev/events/event"
	"github.com/colindev/events/redis"
)

func TestListener(t *testing.T) {
	l := New(redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", "127.0.0.1:6379")
	}, 10))

	wg := &sync.WaitGroup{}

	fn := func(ev event.Event, rd event.RawData) {
		t.Log(ev, rd.String())
		if rd.String() != "world" {
			t.Errorf("On(%v, %v)", ev, rd.String())
		}
		wg.Done()
	}

	l.On(event.Event("hello.*"), fn, fn, fn)

	wg.Add(3)

	l.Trigger(event.Event("hello.a"), event.RawData("world"))

	wg.Wait()
}
