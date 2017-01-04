package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/colindev/events/event"
)

const (

	// CAuth 登入名稱流前綴
	CAuth byte = '$'
	// CEvent 事件長度流前綴
	CEvent byte = '='
	// CAddChan 註冊頻道流前綴
	CAddChan byte = '+'
	// CDelChan 移除頻道流前綴
	CDelChan byte = '-'
	// CErr 錯誤訊息流前綴
	CErr byte = '!'
	// CReply 回應流前綴
	CReply byte = '*'
	// CPing client ping
	CPing byte = '@'
	// CPong reply ping
	CPong byte = '@'
	// CRecover client 請求過往資料
	CRecover byte = '>'
	// CTarget specify receiver
	CTarget byte = '<'
	// CInfo 請求 server 資料
	CInfo byte = '#'

	// Writable flag
	Writable = 1
	// Readable flag
	Readable = 2
)

var (
	OK  = []byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x9, 0x6e, 0x88, 0x0, 0xff}
	EOL = []byte{'\r', '\n'}
)

// Flag is Read Write flag
type Flag int

func (i Flag) String() string {
	var w, r string
	if i&Readable == Readable {
		r = "r"
	} else {
		r = "-"
	}
	if i&Writable == Writable {
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
	// 數字轉 []byte 用
	lenBox  [32]byte
	pending int
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

func (c *conn) readLine() ([]byte, error) {
	b, err := c.r.ReadSlice('\n')
	if err != nil {
		return nil, err
	}

	i := len(b) - 2
	if i < 0 || b[i] != '\r' {
		return nil, errors.New("empty line")
	}

	return b[:i], nil
}

func (c *conn) readLen(p []byte) ([]byte, error) {
	n, err := ParseLen(p)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	_, err = io.ReadFull(c.r, buf)
	if err != nil {
		return nil, err
	}

	// 取出後面換行
	c.readLine()
	return buf, nil
}

func (c *conn) Receive() (ret interface{}, err error) {

	line, err := c.readLine()
	// TODO 隔離測試時 conn 是 nil
	if err != nil {
		return
	}
	// log.Printf("<- client [%s]: %v = [%s] %v\n", c.name, line, line, err)

	switch line[0] {
	case CReply:
		ret = &Reply{strings.TrimSpace(string(line[1:]))}

	case CErr:
		p, e := c.readLen(line[1:])
		if e != nil {
			err = e
			return
		}
		return nil, errors.New(string(p))

	case CPong:
		p, e := c.readLen(line[1:])
		if e != nil {
			err = e
			return
		}

		ret = &Event{
			Name: event.PONG,
			Data: p,
		}

	case CEvent:
		p, e := c.readLen(line[1:])
		if e != nil {
			err = e
			return
		}

		eventName, eventData, e := ParseEvent(p)
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

func (c *conn) writeLen(prefix byte, n int) error {
	c.w.WriteByte(prefix)
	c.w.WriteString(strconv.Itoa(n))
	_, err := c.w.Write(EOL)
	return err
}

func (c *conn) writeTargetAndLen(prefix byte, target string, n int) error {
	c.w.WriteByte(prefix)
	c.w.WriteString(target)
	c.w.WriteByte(':')
	c.w.WriteString(strconv.Itoa(n))
	_, err := c.w.Write(EOL)
	return err
}

func (c *conn) writeEvent(p []byte) error {
	c.writeLen(CEvent, len(p))
	_, err := c.w.Write(p)
	return err
}

func (c *conn) writeEventTo(name string, p []byte) error {
	c.writeTargetAndLen(CTarget, name, len(p))
	_, err := c.w.Write(p)
	return err
}

func (c *conn) Auth(flags int) error {
	c.w.WriteByte(CAuth)
	c.w.WriteString(fmt.Sprintf("%s:%d", c.name, flags))
	return c.flush()
}

func (c *conn) Recover(since, until int64) error {
	c.w.WriteByte(CRecover)
	c.w.WriteString(strconv.FormatInt(since, 10) + ":" + strconv.FormatInt(until, 10))
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
	rd, err := event.Compress(rd)
	if err != nil {
		return err
	}
	c.writeEvent(MakeEventStream(ev, rd))
	return c.flush()
}

func (c *conn) FireTo(name string, ev event.Event, rd event.RawData) error {
	rd, err := event.Compress(rd)
	if err != nil {
		return err
	}
	c.writeEventTo(name, MakeEventStream(ev, rd))
	return c.flush()
}

func (c *conn) Ping(m string) error {
	c.writeLen(CPing, len(m))
	c.w.WriteString(m)

	return c.flush()
}

func (c *conn) Info() error {
	c.w.WriteByte(CInfo)

	return c.flush()
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

func (c *conn) Err() error {
	return c.err
}
