package listener

import (
	"math/rand"
	"testing"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
)

func TestListener(t *testing.T) {
	l := New(func() (client.Conn, error) { return nil, nil })

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
