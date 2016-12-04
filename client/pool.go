package client

import (
	"container/list"
	"errors"
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

func (p *pool) put(c Conn) error {
	// TODO 處理錯誤連線
	// TODO 評估處理超時連線
	p.mu.Lock()

	if c == nil {
		p.mu.Unlock()
		return errors.New("conn is nil")
	}

	p.list.PushFront(c)
	if p.list.Len() > p.maxIdle {
		c = p.list.Remove(p.list.Back()).(Conn)
	} else {
		c = nil
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

// 只能 Fire, Close, Receive
// 不處理其他方法,省略清除原本通訊設定
type maskConn struct {
	p *pool
	c Conn
}

func (m *maskConn) Fire(ev event.Event, rd event.RawData) error {
	return m.c.Fire(ev, rd)
}
func (m *maskConn) Receive() (interface{}, error) {
	return m.c.Receive()
}
func (m *maskConn) Close() error {
	return m.p.put(m.c)
}
func (m *maskConn) Auth() error {
	return errors.New("pooled conn not support Auth()")
}
func (m *maskConn) Recover() error {
	return errors.New("pooled conn not support Recover()")
}
func (m *maskConn) RecoverSince(int64) error {
	return errors.New("pooled conn not support RecoverSince()")
}
func (m *maskConn) Subscribe(...string) error {
	return errors.New("pooled conn not support Subscribe()")
}
func (m *maskConn) Unsubscribe(...string) error {
	return errors.New("pooled conn not support Unsubscribe()")
}

// errConn
type errConn struct{ err error }

func (err *errConn) Fire(event.Event, event.RawData) error { return err.err }
func (err *errConn) Receive() (interface{}, error)         { return nil, err.err }
func (err *errConn) Close() error                          { return err.err }
func (err *errConn) Auth() error                           { return err.err }
func (err *errConn) Recover() error                        { return err.err }
func (err *errConn) RecoverSince(int64) error              { return err.err }
func (err *errConn) Subscribe(...string) error             { return err.err }
func (err *errConn) Unsubscribe(...string) error           { return err.err }
