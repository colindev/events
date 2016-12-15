package listener

import (
	"bufio"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
)

type fConn struct {
	r *bufio.Reader
}

func (f *fConn) Receive() (interface{}, error) {
	return f.r.ReadSlice('\n')
}
func (f *fConn) Close() error                          { return nil }
func (f *fConn) Recover(int64, int64) error            { return nil }
func (f *fConn) Auth(int) error                        { return nil }
func (f *fConn) Subscribe(...string) error             { return nil }
func (f *fConn) Unsubscribe(...string) error           { return nil }
func (f *fConn) Fire(event.Event, event.RawData) error { return nil }
func (f *fConn) Ping(string) error                     { return nil }
func (f *fConn) Conn() net.Conn                        { return nil }
func (f *fConn) Err() error                            { return nil }

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

	l.WaitHandler()
}

func TestListener_RunForever(t *testing.T) {

	var (
		trigged int
		quit    = make(chan bool, 1)
	)

	dial := func() (client.Conn, error) {
		t.Logf("disconn(%d) Dialing...", trigged)
		if trigged >= 3 {
			t.Log("dial signal quit")
			quit <- true
		}
		return &fConn{r: bufio.NewReader(strings.NewReader(""))}, nil
	}

	l := New(dial).On(event.Disconnected, func(ev event.Event, rd event.RawData) {
		t.Logf("%s: %s", ev, rd)
		trigged++
	})

	l.RunForever(quit, time.Millisecond*50).WaitHandler()
	// +-+-+-+-
	if trigged != 4 {
		t.Error("reconn:", trigged)
	}
}
