package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/colindev/events/server/store"
)

var (
	// CAuth 登入名稱流前綴
	CAuth byte = '$'
	// CEvent 事件長度流前綴
	CEvent byte = '='
	// CAddChan 註冊頻道流前綴
	CAddChan byte = '+'
	// CDelChan 移除頻道流前綴
	CDelChan byte = '-'
	// CError 錯誤訊息流前綴
	CError byte = '!'
	// CReply 回應流前綴
	CReply byte = '*'
	// CPing client ping
	CPing byte = '@'
	// CPong reply ping
	CPong byte = '@'
	// CRecover client 請求過往資料
	CRecover byte = '>'

	// EOL 換行bytes
	EOL = []byte{'\r', '\n'}
)

// Conn 包裝 net.Conn (TCP) 連線
type Conn interface {
	SetName(string)
	GetName() string
	HasName() bool
	SetLastAuth(*store.Auth)
	GetLastAuth() *store.Auth
	RemoteAddr() string
	GetAuth() *store.Auth
	EachChannels(fs ...func(string, *regexp.Regexp) string) map[string]*regexp.Regexp
	ReadLine() ([]byte, error)
	ReadLen([]byte) ([]byte, error)
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

type conn struct {
	*sync.RWMutex
	conn net.Conn
	w    *bufio.Writer
	r    *bufio.Reader
	err  error
	chs  map[string]*regexp.Regexp
	// recover events 用
	lastAuth    *store.Auth
	connectedAt int64
	name        string
	// 數字轉 []byte 用
	lenBox [32]byte
}

func newConn(c net.Conn, t time.Time) Conn {
	return &conn{
		RWMutex:     &sync.RWMutex{},
		conn:        c,
		w:           bufio.NewWriter(c),
		r:           bufio.NewReader(c),
		chs:         map[string]*regexp.Regexp{},
		connectedAt: t.Unix(),
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

// GetAuth 回傳當前連線的登入紀錄
func (c *conn) GetAuth() *store.Auth {
	return &store.Auth{
		Name:        c.name,
		IP:          c.RemoteAddr(),
		ConnectedAt: c.connectedAt,
	}
}

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

func (c *conn) ReadLen(p []byte) ([]byte, error) {
	n, err := parseLen(p)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	_, err = io.ReadFull(c.r, buf)
	if err != nil {
		return nil, err
	}
	// 讀取最後的換行
	c.ReadLine()
	return buf, nil
}

func (c *conn) Subscribe(p []byte) (string, error) {

	c.Lock()
	defer c.Unlock()

	// 去除空白/換行
	ind := strings.TrimSpace(string(p))
	// 判斷是否重複註冊
	if _, exists := c.chs[ind]; exists {
		// 容錯重複註冊
		return ind, nil
	}
	// 轉換 . => \.
	pattem := strings.Replace(ind, ".", "\\.", -1)
	// 轉換 * => [^.]*
	pattem = strings.Replace(pattem, "*", "[^.]*", -1)
	// 建構正規表達式物件
	re, err := regexp.Compile("^" + pattem + "$")
	if err != nil {
		return ind, err
	}

	c.chs[ind] = re
	return ind, nil
}

func (c *conn) Unsubscribe(p []byte) (string, error) {
	c.Lock()
	defer c.Unlock()

	// 去除空白/換行
	ind := strings.TrimSpace(string(p))
	// 判斷是否存在
	if _, exists := c.chs[ind]; !exists {
		return "", fmt.Errorf("channel(%s) not found", ind)
	}

	delete(c.chs, ind)

	return ind, nil
}

func (c *conn) IsListening(eventName string) bool {
	c.RLock()
	defer c.RUnlock()

	ev := []byte(eventName)

	for _, re := range c.chs {
		if re.Match(ev) {
			return true
		}
	}

	return false
}

func (c *conn) EachChannels(fs ...func(string, *regexp.Regexp) string) map[string]*regexp.Regexp {
	c.RLock()
	defer c.RUnlock()

	ret := map[string]*regexp.Regexp{}

NEXT:
	for ch, re := range c.chs {
		for _, f := range fs {
			ch = f(ch, re)
			if ch == "" {
				continue NEXT
			}
		}
		ret[ch] = re
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
	c.w.Write(EOL)
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
	_, err := c.w.Write(EOL)
	return err
}

func (c *conn) writeError(err error) error {
	c.w.WriteByte(CError)
	_, err = c.w.WriteString(err.Error())
	return err
}

func (c *conn) writeReply(m string) error {
	c.w.WriteByte(CReply)
	_, err := c.w.WriteString(m)
	return err
}

func (c *conn) writePong(ping []byte) error {
	c.writeLen(CPong, len(ping))
	_, err := c.w.Write(ping)
	return err
}

func (c *conn) writeEvent(e string) error {
	c.writeLen(CEvent, len(e))
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
