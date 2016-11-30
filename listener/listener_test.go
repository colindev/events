package listener

import (
	"math/rand"
	"testing"
	"time"

	"github.com/colindev/events/event"
	"github.com/colindev/events/redis"
)

func TestListener(t *testing.T) {
	l := New(redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", "127.0.0.1:6379")
	}, 10))

	fn := func(ev event.Event, rd event.RawData) {
		rand.Seed(int64(time.Now().Nanosecond()))
		n := time.Duration(rand.Intn(50))
		t.Log("sleep", n, ev, rd.String())
		if rd.String() != "world" {
			t.Errorf("On(%v, %v)", ev, rd.String())
		}

		time.Sleep(time.Millisecond * n)
	}

	l.On(event.Event("hello.*"), fn, fn, fn)

	l.Trigger(event.Event("hello.a"), event.RawData("world"))

	l.Stop()
}
