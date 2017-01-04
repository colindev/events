package client

import (
	"container/list"
	"errors"
	"net"
	"sync"

	"github.com/colindev/events/event"
)

// Pool 連線池
type Pool interface {
	Get() Conn
	ActiveConn() int
	MaxActive(int)
}

type pool struct {
	dial       func() (Conn, error)
	mu         sync.Mutex
	cond       *sync.Cond
	list       list.List
	maxIdle    int
	maxActive  int
	activeConn int
}

// NewPool return pool instance
func NewPool(dial func() (Conn, error), maxIdle int) Pool {
	return &pool{
		dial:    dial,
		maxIdle: maxIdle,
	}
}

func (p *pool) MaxActive(n int) {
	if n <= 0 {
		return
	}

	p.mu.Lock()
	p.maxActive = n
	p.mu.Unlock()
}

func (p *pool) Get() Conn {

	c, err := p.get()
	if err != nil {
		return &errConn{err}
	}

	return &maskConn{p: p, c: c}
}

func (p *pool) ActiveConn() int {
	p.mu.Lock()
	n := p.activeConn
	p.mu.Unlock()

	return n
}

func (p *pool) release() {
	p.activeConn--
	if p.cond != nil {
		p.cond.Signal()
	}
}

func (p *pool) get() (Conn, error) {

	p.mu.Lock()

	for {

		for i, n := 0, p.list.Len(); i < n; i++ {
			el := p.list.Front()
			// init or return
			if el == nil {
				break
			}

			c := el.Value.(Conn)
			p.list.Remove(el)
			p.mu.Unlock()
			return c, nil
		}

		if p.maxActive == 0 || p.activeConn < p.maxActive {
			p.activeConn++
			dial := p.dial
			p.mu.Unlock()
			c, err := dial()
			if err != nil {
				p.mu.Lock()
				p.release()
				p.mu.Unlock()
				c = nil
			}
			return c, err
		}

		if p.cond == nil {
			p.cond = sync.NewCond(&p.mu)
		}
		p.cond.Wait()
	}
}

// TODO 依照後續的使用狀況觀察評估
// 是否還原連線的初始狀態
// 作法有
// 1. maxIdle 設定為 0, 讓連線池每次都重新建立連線
// 2. 回收連線後先把 flags 設定成 read only 或 0
// 3. 送個 reset 給 server 處理
func (p *pool) put(c Conn) error {
	// TODO 評估處理超時連線
	p.mu.Lock()

	if err := c.Err(); err == nil {
		p.list.PushFront(c)
		if p.list.Len() > p.maxIdle {
			c = p.list.Remove(p.list.Back()).(Conn)
		} else {
			c = nil
		}
	}

	if c == nil {
		if p.cond != nil {
			p.cond.Signal()
		}
		p.mu.Unlock()
		return nil
	}

	p.release()
	p.mu.Unlock()
	return c.Close()
}

// 只能 Auth, Fire, FireTo, Close, Receive, Ping, Info
// 不處理其他方法,省略清除原本通訊設定
type maskConn struct {
	p *pool
	c Conn
}

func (m *maskConn) Fire(ev event.Event, rd event.RawData) error {
	return m.c.Fire(ev, rd)
}
func (m *maskConn) FireTo(name string, ev event.Event, rd event.RawData) error {
	return m.c.FireTo(name, ev, rd)
}
func (m *maskConn) Ping(s string) error {
	return m.c.Ping(s)
}
func (m *maskConn) Info() error {
	return m.c.Info()
}
func (m *maskConn) Receive() (interface{}, error) {
	return m.c.Receive()
}
func (m *maskConn) Auth(i int) error {
	return m.c.Auth(i)
}
func (m *maskConn) Close() error {
	return m.p.put(m.c)
}
func (m *maskConn) Recover(int64, int64) error {
	return errors.New("pooled conn not support RecoverSince()")
}
func (m *maskConn) Subscribe(...string) error {
	return errors.New("pooled conn not support Subscribe()")
}
func (m *maskConn) Unsubscribe(...string) error {
	return errors.New("pooled conn not support Unsubscribe()")
}
func (m *maskConn) Conn() net.Conn {
	return m.c.Conn()
}
func (m *maskConn) Err() error {
	return m.c.Err()
}

// errConn
type errConn struct{ err error }

func (err *errConn) Fire(event.Event, event.RawData) error           { return err.err }
func (err *errConn) FireTo(string, event.Event, event.RawData) error { return err.err }
func (err *errConn) Receive() (interface{}, error)                   { return nil, err.err }
func (err *errConn) Close() error                                    { return err.err }
func (err *errConn) Auth(int) error                                  { return err.err }
func (err *errConn) Ping(string) error                               { return err.err }
func (err *errConn) Info() error                                     { return err.err }
func (err *errConn) Recover(int64, int64) error                      { return err.err }
func (err *errConn) Subscribe(...string) error                       { return err.err }
func (err *errConn) Unsubscribe(...string) error                     { return err.err }
func (err *errConn) Conn() net.Conn                                  { return nil }
func (err *errConn) Err() error                                      { return err.err }
