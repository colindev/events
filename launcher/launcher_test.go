package launcher

import (
	"errors"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
)

type fake struct {
	fn func(...interface{})
}

func (m *fake) Auth(i int) error {
	m.fn("Auth", i)
	return nil
}
func (m *fake) Fire(ev event.Event, rd event.RawData) error {
	m.fn("Fire", ev, rd)
	return nil
}
func (m *fake) Ping(s string) error {
	m.fn("Ping", s)
	return nil
}
func (m *fake) Close() error {
	m.fn("Close")
	return nil
}
func (m *fake) Receive() (interface{}, error) {
	m.fn("Receive")
	return "ok", nil
}
func (m *fake) Recover(since, until int64) error {
	m.fn("Recover", since, until)
	return nil
}
func (m *fake) Subscribe(chs ...string) error {
	m.fn("Subscribe", chs)
	return nil
}
func (m *fake) Unsubscribe(chs ...string) error {
	m.fn("Unsubscribe", chs)
	return nil
}
func (m *fake) Conn() net.Conn {
	m.fn("Conn")
	return nil
}
func (m *fake) Err() error {
	m.fn("Err")
	rand.Seed(int64(time.Now().Nanosecond()))
	if rand.Intn(30)%2 == 0 {
		return errors.New("fake test error")
	}
	return nil
}

func TestForge(t *testing.T) {

	n := 0
	// 僅簡單的檢查一下建構語法
	l := New(client.NewPool(func() (client.Conn, error) {
		return &fake{fn: func(v ...interface{}) {
			if v[0] == "Fire" {
				n++
			}
			t.Log(v...)
		}}, nil
	}, 10))

	l.Fire("", nil)
	l.Fire("", nil)
	l.Fire("", nil)
	l.Fire("", nil)
	l.Fire("", nil)
	l.Fire("", nil)
	l.Fire("", nil)
	l.Close()

	if n != 7 {
		t.Error("fire miss")
	}
}

var benchData = event.RawData("")

func delayConnPoolLauncher(delay time.Duration, i int) Launcher {

	pool := client.NewPool(func() (client.Conn, error) {
		return &fake{fn: func(v ...interface{}) {
			if v[0] == "Fire" {
				time.Sleep(delay)
			}
		}}, nil
	}, 1)
	pool.MaxActive(i)
	return New(pool)
}

func BenchmarkFireConn20delay50nano(b *testing.B) {
	delay := 50
	active := 20
	l := delayConnPoolLauncher(time.Nanosecond*time.Duration(delay), active)
	for i := 0; i < b.N; i++ {
		l.Fire("test.event", benchData)
	}

	l.Close()
}

func BenchmarkFireConn50delay50nano(b *testing.B) {
	delay := 50
	active := 50
	l := delayConnPoolLauncher(time.Nanosecond*time.Duration(delay), active)
	for i := 0; i < b.N; i++ {
		l.Fire("test.event", benchData)
	}

	l.Close()
}
