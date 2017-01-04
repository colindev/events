package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/store"
)

// Conn 包裝 net.Conn (TCP) 連線
type Conn interface {
	SetName(string)
	GetName() string
	HasName() bool
	SetFlags(int)
	Writable() bool
	Readable() bool
	SetAuthed(bool)
	IsAuthed() bool
	SetLastAuth(*store.Auth)
	GetLastAuth() *store.Auth
	RemoteAddr() string
	GetAuth() *store.Auth
	EachChannels(fs ...func(event.Event) event.Event) map[event.Event]bool
	ReadLine() ([]byte, error)
	ReadLen([]byte) ([]byte, error)
	ReadTargetAndLen([]byte) (string, []byte, error)
	Subscribe([]byte) (string, error)
	Unsubscribe([]byte) (string, error)
	IsListening(string) bool
	Err() error
	Close(error) error
	SendError(error) error
	SendReply(string) error
	SendPong([]byte) error
	SendEvent(e string) error
}

// ConnStatus contain conn status
type ConnStatus struct {
	Channel  []event.Event
	Name     string
	LastAuth *store.Auth `json:",omitempty"`
	Flag     int
}

type conn struct {
	sync.RWMutex
	conn   net.Conn
	w      *bufio.Writer
	r      *bufio.Reader
	flags  int
	err    error
	chs    map[event.Event]bool
	authed bool
	// recover events 用
	lastAuth    *store.Auth
	connectedAt int64
	name        string
	// 數字轉 []byte 用
	lenBox [32]byte
}

func newConn(c net.Conn, t time.Time) Conn {
	return &conn{
		conn:        c,
		w:           bufio.NewWriter(c),
		r:           bufio.NewReader(c),
		chs:         map[event.Event]bool{},
		connectedAt: t.Unix(),
	}
}

// Status non-export
func (c *conn) Status(ignoreWriteOnly bool) *ConnStatus {
	c.RLock()
	name := c.name
	flags := c.flags
	lastAuth := c.lastAuth
	c.RUnlock()

	// NOTE ignore write only (launcher conn)
	if ignoreWriteOnly && flags == client.Writable {
		return nil
	}

	chs := []event.Event{}
	c.EachChannels(func(ev event.Event) event.Event {
		chs = append(chs, ev)
		return ev
	})

	return &ConnStatus{
		Channel:  chs,
		Name:     name,
		LastAuth: lastAuth,
		Flag:     flags,
	}
}

// SetLastAuth 注入上一次登入紀錄
func (c *conn) SetLastAuth(a *store.Auth) {
	c.lastAuth = a
}
func (c *conn) GetLastAuth() *store.Auth {
	return c.lastAuth
}

func (c *conn) SetName(name string) {
	c.name = name
}
func (c *conn) GetName() string {
	return c.name
}
func (c *conn) HasName() bool {
	return c.name != ""
}

func (c *conn) RemoteAddr() string {
	if c.conn == nil {
		return ""
	}
	return c.conn.RemoteAddr().String()
}

func (c *conn) SetFlags(flags int) {
	// 如果 client 不指定讀取權限
	// 就轉導 write buffer 到 ioutil.Discard
	if flags&client.Readable > 0 {
		c.w = bufio.NewWriter(c.conn)
	} else {
		c.w = bufio.NewWriter(ioutil.Discard)
	}

	c.flags = flags
	// writeable 僅略過處理 event 資料
	// 其餘的照常處理
	// 由 Hub.handle 處理
}

func (c *conn) Writable() bool {
	return (c.flags & client.Writable) != 0
}

func (c *conn) Readable() bool {
	return (c.flags & client.Readable) != 0
}

func (c *conn) SetAuthed(t bool) {
	c.authed = t
}

func (c *conn) IsAuthed() bool {
	return c.authed
}

// GetAuth 回傳當前連線的登入紀錄
func (c *conn) GetAuth() *store.Auth {
	return &store.Auth{
		Name:        c.name,
		IP:          c.RemoteAddr(),
		ConnectedAt: c.connectedAt,
	}
}

// ReadLine 回傳去除結尾換行符號後的bytes
func (c *conn) ReadLine() ([]byte, error) {
	b, err := c.r.ReadSlice('\n')
	if err != nil {
		return nil, err
	}

	i := len(b) - 2
	if i < 0 {
		return nil, nil
	} else if b[i] != '\r' {
		i = i + 1
	}

	return b[:i], nil
}

func (c *conn) ReadLen(p []byte) (b []byte, err error) {
	var n int64
	n, err = client.ParseLen(p)
	if err != nil {
		return
	}
	b = make([]byte, n)
	_, err = io.ReadFull(c.r, b)
	if err != nil {
		return
	}
	// 讀取最後的換行
	c.ReadLine()
	return
}

func (c *conn) ReadTargetAndLen(p []byte) (target string, b []byte, err error) {
	var n int64
	target, n, err = client.ParseTargetAndLen(p)
	if err != nil {
		return
	}

	b = make([]byte, n)
	_, err = io.ReadFull(c.r, b)
	if err != nil {
		return
	}

	c.ReadLine()
	return
}

func (c *conn) Subscribe(p []byte) (string, error) {

	c.Lock()
	defer c.Unlock()

	// 去除空白/換行
	ind := strings.TrimSpace(string(p))
	ev := event.Event(ind)
	// 判斷是否重複註冊
	if _, exists := c.chs[ev]; exists {
		// 容錯重複註冊
		return ind, nil
	}

	c.chs[ev] = true
	return ind, nil
}

func (c *conn) Unsubscribe(p []byte) (string, error) {
	c.Lock()
	defer c.Unlock()

	// 去除空白/換行
	ind := strings.TrimSpace(string(p))
	ev := event.Event(ind)
	// 判斷是否存在
	if _, exists := c.chs[ev]; !exists {
		return "", fmt.Errorf("channel(%s) not found", ind)
	}

	delete(c.chs, ev)

	return ind, nil
}

func (c *conn) IsListening(eventName string) bool {
	c.RLock()
	defer c.RUnlock()

	ev := event.Event(eventName)

	for ch := range c.chs {
		if ch.Match(ev) {
			return true
		}
	}

	return false
}

func (c *conn) EachChannels(fs ...func(event.Event) event.Event) map[event.Event]bool {
	c.RLock()
	defer c.RUnlock()

	ret := map[event.Event]bool{}

NEXT:
	for ch := range c.chs {
		for _, f := range fs {
			ch = f(ch)
			if ch.String() == "" {
				continue NEXT
			}
		}
		ret[ch] = true
	}

	return ret
}

func (c *conn) Err() error {
	return c.err
}

func (c *conn) Close(err error) error {
	c.Lock()
	defer c.Unlock()
	if c.err == nil {
		c.err = err
		c.conn.Close()
	}
	return err
}

func (c *conn) flush() error {
	c.w.Write(client.EOL)
	if err := c.w.Flush(); err != nil {
		return c.Close(err)
	}

	return nil
}

func (c *conn) writeLen(prefix byte, n int) error {

	i := len(c.lenBox) - 1
	for {
		c.lenBox[i] = byte('0' + n%10)
		i--
		n = n / 10
		if n == 0 {
			break
		}
	}

	c.w.WriteByte(prefix)
	c.w.Write(c.lenBox[i+1:])
	_, err := c.w.Write(client.EOL)
	return err
}

func (c *conn) writeReply(m string) error {
	c.w.WriteByte(client.CReply)
	_, err := c.w.WriteString(m)
	return err
}

func (c *conn) writeError(err error) error {
	c.writeLen(client.CErr, len(err.Error()))
	_, err = c.w.WriteString(err.Error())
	return err
}

func (c *conn) writePong(ping []byte) error {
	c.writeLen(client.CPong, len(ping))
	_, err := c.w.Write(ping)
	return err
}

func (c *conn) writeEvent(e string) error {
	c.writeLen(client.CEvent, len(e))
	_, err := c.w.WriteString(e)
	return err
}

func (c *conn) SendError(err error) error {
	c.writeError(err)
	return c.flush()
}

func (c *conn) SendReply(m string) error {
	c.writeReply(m)
	return c.flush()
}

func (c *conn) SendPong(ping []byte) error {
	c.writePong(ping)
	return c.flush()
}

func (c *conn) SendEvent(e string) error {
	c.writeEvent(e)
	return c.flush()
}
