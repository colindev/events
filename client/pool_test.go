package client

import (
	"errors"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/colindev/events/event"
)

type fake struct {
	fn func(...interface{})
}

func (m *fake) Auth(i int) error {
	m.fn(m.Auth, i)
	return nil
}
func (m *fake) Fire(ev event.Event, rd event.RawData) error {
	m.fn(m.Fire, ev, rd)
	return nil
}
func (m *fake) Ping(s string) error {
	m.fn(m.Ping, s)
	return nil
}
func (m *fake) Info() error {
	m.fn(m.Info)
	return nil
}
func (m *fake) Close() error {
	m.fn(m.Close)
	return nil
}
func (m *fake) Receive() (interface{}, error) {
	m.fn(m.Receive)
	return "ok", nil
}
func (m *fake) Recover(since, until int64) error {
	m.fn(m.Recover, since, until)
	return nil
}
func (m *fake) Subscribe(chs ...string) error {
	m.fn(m.Subscribe, chs)
	return nil
}
func (m *fake) Unsubscribe(chs ...string) error {
	m.fn(m.Unsubscribe, chs)
	return nil
}
func (m *fake) Conn() net.Conn {
	return nil
}
func (m *fake) Err() error {
	m.fn(m.Err)
	rand.Seed(int64(time.Now().Nanosecond()))
	if rand.Intn(30)%2 == 0 {
		return errors.New("fake test error")
	}
	return nil
}

func TestPool(t *testing.T) {

	p := NewPool(func() (Conn, error) {
		return &fake{
			fn: func(v ...interface{}) {},
		}, nil
	}, 3)

	p.MaxActive(10)

	n := 30
	run := make(chan event.Event, n)
	end := make(chan event.Event, n)

	for i := 0; i < n; i++ {
		go func(i int) {
			c := p.Get()
			ev := event.Event("x." + strconv.Itoa(i))
			run <- ev
			time.Sleep(time.Second * 3)
			t.Log("close error:", c.Close())
			end <- ev
		}(i)
	}

	for i := 0; i < n; i++ {
		t.Log("run:", <-run)
	}
	for i := 0; i < n; i++ {
		t.Log("end:", <-end)
	}
}
