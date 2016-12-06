package client

import (
	"strconv"
	"testing"
	"time"

	"github.com/colindev/events/event"
)

type fake struct{ n int }

func (m *fake) Fire(ev event.Event, rd event.RawData) error {
	return nil
}
func (m *fake) Receive() (interface{}, error) {
	return "ok", nil
}
func (m *fake) Close() error {
	return nil
}
func (m *fake) Auth() error {
	return nil
}
func (m *fake) Ping(string) error {
	return nil
}
func (m *fake) Recover() error {
	return nil
}
func (m *fake) RecoverSince(int64) error {
	return nil
}
func (m *fake) Subscribe(...string) error {
	return nil
}
func (m *fake) Unsubscribe(...string) error {
	return nil
}

func TestPool(t *testing.T) {

	p := NewPool(func() (Conn, error) {
		return &fake{}, nil
	}, 3)

	p.MaxActive(10)

	n := 30
	run := make(chan event.Event, n)
	end := make(chan event.Event, n)

	for i := 0; i < n; i++ {
		go func(i int) {
			c := p.Get()
			c.(*maskConn).c.(*fake).n = i
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
