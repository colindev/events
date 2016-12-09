package client

import (
	"fmt"
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

func Example_maskConn() {
	p := NewPool(func() (Conn, error) {
		return &fake{
			fn: func(v ...interface{}) {
				fmt.Printf("%T %v\n", v[0], v[1:])
			},
		}, nil
	}, 1)

	// masked
	p.Get().Close()
	p.Get().Recover(2, 3)
	p.Get().Subscribe("x", "y", "z")
	p.Get().Unsubscribe("x", "y", "z")

	// adapted
	p.Get().Auth(1)
	p.Get().Fire("a", []byte{'b'})
	p.Get().Ping("pong")
	p.Get().Receive()
	// output:
	// func(int) error [1]
	// func(event.Event, event.RawData) error [a b]
	// func(string) error [pong]
	// func() (interface {}, error) []
}

func TestPool(t *testing.T) {

	p := NewPool(func() (Conn, error) {
		return &fake{fn: func(v ...interface{}) {}}, nil
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
