package client

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/colindev/events/connection"
	"github.com/colindev/events/event"
)

// Flag is Read Write flag
type Flag int

func (i Flag) String() string {
	var w, r string
	if i&connection.Readable == connection.Readable {
		r = "r"
	} else {
		r = "-"
	}
	if i&connection.Writable == connection.Writable {
		w = "w"
	} else {
		w = "-"
	}
	return fmt.Sprintf("%s%s", r, w)
}

// Reply 包裝回應內容
type Reply struct {
	s string
}

func (r *Reply) String() string {
	return r.s
}

// Event 包裝事件名稱跟資料
type Event struct {
	Target string // 定傳送目標
	Name   event.Event
	Data   event.RawData
}

// Conn 包裝 net.Conn
type Conn interface {
	Close() error
	Recover(int64, int64) error
	Auth(int) error
	Subscribe(...string) error
	Unsubscribe(...string) error
	Fire(event.Event, event.RawData) error
	FireTo(string, event.Event, event.RawData) error
	Ping(string) error
	Info() error
	Receive() (interface{}, error)
	Conn() net.Conn
	Err() error
}

type conn struct {
	*sync.Mutex
	conn net.Conn
	name string
	w    *bufio.Writer
	r    *bufio.Reader
	err  error
}

// Dial 回傳 conn 實體物件
func Dial(name, addr string) (Conn, error) {

	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &conn{
		Mutex: &sync.Mutex{},
		conn:  c,
		name:  name,
		w:     bufio.NewWriter(c),
		r:     bufio.NewReader(c),
	}, nil
}

func (c *conn) Conn() net.Conn {
	return c.conn
}

func (c *conn) Close() error {
	return c.conn.Close()
}

func (c *conn) Receive() (ret interface{}, err error) {

	line, err := connection.ReadLine(c.r)
	// TODO 隔離測試時 conn 是 nil
	if err != nil {
		return
	}
	// log.Printf("<- client [%s]: %v = [%s] %v\n", c.name, line, line, err)

	switch line[0] {
	case connection.CReply:
		ret = &Reply{strings.TrimSpace(string(line[1:]))}

	case connection.CErr:
		p, e := connection.ReadLen(c.r, line[1:])
		if e != nil {
			err = e
			return
		}
		return nil, errors.New(string(p))

	case connection.CPong:
		p, e := connection.ReadLen(c.r, line[1:])
		if e != nil {
			err = e
			return
		}

		ret = &Event{
			Name: event.PONG,
			Data: p,
		}

	case connection.CEvent:
		p, e := connection.ReadLen(c.r, line[1:])
		if e != nil {
			err = e
			return
		}

		eventName, eventData, e := connection.ParseEvent(p)
		if e != nil {
			err = e
			return
		}

		// 解壓縮
		b, e := event.Uncompress(eventData)
		if e != nil {
			err = e
			return
		}
		ret = &Event{
			Name: event.Event(eventName),
			Data: b,
		}
	}

	return ret, err
}

func (c *conn) Auth(flags int) error {
	connection.WriteAuth(c.w, c.name, flags)
	return c.flush(connection.EOL)
}

func (c *conn) Recover(since, until int64) error {
	connection.WriteRecover(c.w, since, until)
	return c.flush(connection.EOL)
}

func (c *conn) Subscribe(chans ...string) error {
	connection.WriteSubscribe(c.w, chans...)
	return c.flush(nil)
}

func (c *conn) Unsubscribe(chans ...string) error {
	connection.WriteUnsubscribe(c.w, chans...)
	return c.flush(nil)
}

func (c *conn) Fire(ev event.Event, rd event.RawData) error {
	rd, err := event.Compress(rd)
	if err != nil {
		return err
	}
	connection.WriteEvent(c.w, connection.MakeEventStream(ev, rd))
	return c.flush(connection.EOL)
}

func (c *conn) FireTo(name string, ev event.Event, rd event.RawData) error {
	rd, err := event.Compress(rd)
	if err != nil {
		return err
	}
	connection.WriteEventTo(c.w, name, connection.MakeEventStream(ev, rd))
	return c.flush(connection.EOL)
}

func (c *conn) Ping(m string) error {
	connection.WritePing(c.w, m)
	return c.flush(connection.EOL)
}

func (c *conn) Info() error {
	connection.WriteInfo(c.w)
	return c.flush(connection.EOL)
}

func (c *conn) flush(p []byte) error {
	c.w.Write(p)
	if err := c.w.Flush(); err != nil {
		c.Lock()
		c.err = err
		c.Unlock()
		return c.Close()
	}

	return nil
}

func (c *conn) Err() error {
	return c.err
}
