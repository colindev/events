package main

import (
	"bufio"
	"fmt"
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
	// CLength 事件長度流前綴
	CLength byte = '='
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
	// CRecover client 請求過往資料
	CRecover byte = '>'

	// OK 成功回應 bytes
	OK = []byte{'O', 'K'}
	// PONG ...
	PONG = []byte{'P', 'O', 'N', 'G'}
	// EOL 換行bytes
	EOL = []byte{'\r', '\n'}
)

// Conn 包裝 net.Conn (TCP) 連線
type Conn struct {
	*sync.RWMutex
	conn     net.Conn
	w        *bufio.Writer
	r        *bufio.Reader
	err      error
	channals map[string]*regexp.Regexp
	// recover events 用
	lastAuth    *store.Auth
	connectedAt int64
	name        string
	// 數字轉 []byte 用
	lenBox [32]byte
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

// SetLastAuth 注入上一次登入紀錄
func (c *Conn) SetLastAuth(a *store.Auth) {
	c.lastAuth = a
}

// GetAuth 回傳當前連線的登入紀錄
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

func (c *Conn) subscribe(p []byte) (string, error) {

	c.Lock()
	defer c.Unlock()

	// 去除空白/換行
	ind := strings.TrimSpace(string(p))
	// 判斷是否重複註冊
	if _, exists := c.channals[ind]; exists {
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

	c.channals[ind] = re
	return ind, nil
}

func (c *Conn) unsubscribe(p []byte) error {
	c.Lock()
	defer c.Unlock()

	// 去除空白/換行
	ind := strings.TrimSpace(string(p))
	// 判斷是否存在
	if _, exists := c.channals[ind]; exists {
		return fmt.Errorf("channal(%s) not found", ind)
	}

	delete(c.channals, ind)

	return nil
}

func (c *Conn) isListening(eventName string) bool {
	c.RLock()
	defer c.RUnlock()

	ev := []byte(eventName)

	for _, re := range c.channals {
		if re.Match(ev) {
			return true
		}
	}

	return false
}

func (c *Conn) close(err error) error {
	c.Lock()
	defer c.Unlock()
	if c.err == nil {
		c.err = err
		c.conn.Close()
	}
	return err
}

func (c *Conn) writeLen(n int) error {

	i := len(c.lenBox) - 1
	for {
		c.lenBox[i] = byte('0' + n%10)
		i--
		n = n / 10
		if n == 0 {
			break
		}
	}

	c.w.WriteByte(CLength)
	c.w.Write(c.lenBox[i+1:])
	_, err := c.w.Write(EOL)
	return err
}

func (c *Conn) writeError(e error) {
	c.w.WriteByte(CError)
	c.w.WriteString(e.Error())
	c.flush()
}

func (c *Conn) writeOk() {
	c.w.WriteByte(CReply)
	c.w.Write(OK)
	c.flush()
}

func (c *Conn) writePong() {
	c.w.WriteByte(CReply)
	c.w.Write(PONG)
	c.flush()
}

func (c *Conn) writeEvent(e string) {
	c.writeLen(len(e))
	c.w.WriteString(e)
	c.flush()
}

func (c *Conn) flush() error {
	c.w.Write(EOL)
	if err := c.w.Flush(); err != nil {
		return c.close(err)
	}

	return nil
}