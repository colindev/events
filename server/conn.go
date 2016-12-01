package main

import (
	"bufio"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/colindev/events/server/store"
)

type HandlerFunc func(c *Conn)

type Conn struct {
	*sync.RWMutex
	conn     net.Conn
	w        *bufio.Writer
	r        *bufio.Reader
	channals map[string]*regexp.Regexp
	// recover events 用
	lastAuth    *store.Auth
	connectedAt int64
	name        string
}

func newConn(conn net.Conn, t time.Time) *Conn {
	return &Conn{
		RWMutex:     &sync.RWMutex{},
		conn:        conn,
		w:           bufio.NewWriter(conn),
		r:           bufio.NewReader(conn),
		channals:    map[string]*regexp.Regexp{},
		connectedAt: t.Unix(),
	}
}

func (c *Conn) SetLastAuth(a *store.Auth) {
	c.lastAuth = a
}

func (c *Conn) GetAuth() *store.Auth {
	return &store.Auth{
		Name:        c.name,
		IP:          c.conn.RemoteAddr().String(),
		ConnectedAt: c.connectedAt,
	}
}

func (c *Conn) readLine() ([]byte, error) {
	return c.r.ReadSlice('\n')
}

func (c *Conn) subscribe(p []byte) error {

	c.Lock()
	defer c.Unlock()

	// 去除空白/換行
	ind := strings.TrimSpace(string(p))
	// 判斷是否重複註冊
	if _, exists := c.channals[ind]; exists {
		// 容錯重複註冊
		return nil
	}
	// 轉換 . => \.
	pattem := strings.Replace(ind, ".", "\\.", -1)
	// 轉換 * => [^.]*
	pattem = strings.Replace(pattem, "*", "[^.]*", -1)
	// 建構正規表達式物件
	re, err := regexp.Compile("^" + pattem + "$")
	if err != nil {
		return err
	}

	c.channals[ind] = re
	return nil
}

func (c *Conn) Test(ev []byte) bool {
	c.RLock()
	defer c.RUnlock()

	for _, re := range c.channals {
		// TODO
		if re.Match(ev) {
			return true
		}
	}

	return false
}

func (c *Conn) close() error {
	return c.conn.Close()
}
