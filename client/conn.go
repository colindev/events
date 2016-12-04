package client

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/colindev/events/event"
)

var (
	CAuth    byte = '$'
	CRecover byte = '>'
	CAddChan byte = '+'
	CDelChan byte = '-'
	CReply   byte = '*'
	CLength  byte = '='

	PING = []byte{'@'}
	EOL  = []byte{'\r', '\n'}
)

// Reply 包裝回應內容
type Reply struct {
	s string
}

func (r *Reply) String() string {
	return r.s
}

// Event 包裝事件名稱跟資料
type Event struct {
	Name event.Event
	Data event.RawData
}

// Conn 包裝 net.Conn
type Conn interface {
	Close() error
	Recover() error
	RecoverSince(int64) error
	Auth() error
	Subscribe(...string) error
	Unsubscribe(...string) error
	Fire(event.Event, event.RawData) error
	Receive() (interface{}, error)
}

type conn struct {
	*sync.Mutex
	conn net.Conn
	name string
	w    *bufio.Writer
	r    *bufio.Reader
	err  error
	// 數字轉 []byte 用
	lenBox  [32]byte
	pending int
}

// New 回傳 conn 實體物件
func New(name, addr string) (Conn, error) {

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

func (c *conn) Close() error {
	return c.conn.Close()
}

func (c *conn) readLine() ([]byte, error) {
	return c.r.ReadSlice('\n')
}

func (c *conn) Receive() (ret interface{}, err error) {

	line, err := c.readLine()
	if err != nil {
		return
	}

	switch line[0] {
	case CReply:
		ret = &Reply{strings.TrimSpace(string(line[1:]))}
	case CLength:
		n, e := parseLen(line[1:])
		if e != nil {
			err = e
			return
		}
		//log.Printf("length: %d\n", n)

		p := make([]byte, n)
		_, e = io.ReadFull(c.r, p)
		//log.Printf("read: %+v = [%s]\n", p, p)
		// 去除換行
		c.readLine()
		if e != nil {
			err = e
			return
		}

		eventName, eventData, e := parseEvent(p)
		if e != nil {
			err = e
			return
		}
		//log.Println("receive:", string(eventName), string(eventData), e)
		if e != nil {
			return
		}
		ret = &Event{
			Name: event.Event(eventName),
			Data: event.RawData(eventData),
		}
	}

	return ret, err
}

func (c *conn) writeLen(n int) error {
	c.w.WriteByte(CLength)
	c.w.WriteString(strconv.Itoa(n))
	_, err := c.w.Write(EOL)
	return err
}

func (c *conn) writeEvent(p []byte) error {
	c.writeLen(len(p))
	c.w.Write(p)
	return c.flush()
}

func (c *conn) Auth() error {
	c.w.WriteByte(CAuth)
	c.w.WriteString(c.name)
	return c.flush()
}

func (c *conn) Recover() error {
	c.w.WriteByte(CRecover)
	return c.flush()
}

func (c *conn) RecoverSince(i int64) error {
	c.w.WriteByte(CRecover)
	c.w.WriteString(strconv.FormatInt(1, 10))
	return c.flush()
}

func (c *conn) Subscribe(chans ...string) error {
	for _, ch := range chans {
		c.w.WriteByte(CAddChan)
		c.w.WriteString(ch)
		c.w.Write(EOL)
	}

	return c.w.Flush()
}

func (c *conn) Unsubscribe(chans ...string) error {
	for _, ch := range chans {
		c.w.WriteByte(CDelChan)
		c.w.WriteString(ch)
		c.w.Write(EOL)
	}

	return c.w.Flush()
}

func (c *conn) Fire(ev event.Event, rd event.RawData) error {
	buf := bytes.NewBuffer(ev.Bytes())
	buf.WriteByte(':')
	buf.Write(rd.Bytes())

	return c.writeEvent(buf.Bytes())
}

func (c *conn) flush() error {
	c.w.Write(EOL)
	if err := c.w.Flush(); err != nil {
		c.Lock()
		c.err = err
		c.Unlock()
		return c.Close()
	}

	return nil
}

func parseLen(p []byte) (int64, error) {

	raw := string(p)

	if len(p) == 0 {
		return -1, errors.New("length error:" + raw)
	}

	var negate bool
	if p[0] == '-' {
		negate = true
		p = p[1:]
		if len(p) == 0 {
			return -1, errors.New("length error:" + raw)
		}
	}

	var n int64
	for _, b := range p {

		if b == '\r' || b == '\n' {
			break
		}
		n *= 10
		if b < '0' || b > '9' {
			return -1, errors.New("not number:" + string(b))
		}

		n += int64(b - '0')
	}

	if negate {
		n = -n
	}

	return n, nil
}

func parseEvent(p []byte) (ev, data []byte, err error) {

	var sp int
	box := [30]byte{}

	for i, b := range p {
		if b == ':' {
			sp = i
			break
		}
		box[i] = b
	}
	ev = box[:sp]
	data = p[sp+1:]

	if len(ev) > 30 {
		err = errors.New("event name over 30 char")
	} else if len(ev) == 0 {
		err = errors.New("event name is empty")
	} else if len(data) == 0 {
		err = errors.New("event data empty")
	}

	return
}
