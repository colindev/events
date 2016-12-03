package main

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/colindev/events/event"
	"github.com/colindev/events/server/store"
)

// Hub 負責管理連線
type Hub struct {
	*sync.RWMutex
	m     map[string]*Conn
	store *store.Store
	*log.Logger
}

// NewHub create and return a Hub instance
func NewHub(env *Env, logger *log.Logger) (*Hub, error) {

	sto, err := store.New(store.Config{
		Debug:      env.Debug,
		AuthDSN:    env.AuthDSN,
		EventDSN:   env.EventDSN,
		GCDuration: env.GCDuration,
	})
	if err != nil {
		return nil, err
	}

	return &Hub{
		RWMutex: &sync.RWMutex{},
		m:       map[string]*Conn{},
		store:   sto,
		Logger:  logger,
	}, nil
}

// Auth 執行登入紀錄
func (h *Hub) auth(c *Conn, p []byte) error {

	h.Lock()
	defer h.Unlock()

	if len(p) <= 1 {
		return errors.New("empty id")
	}

	name := strings.TrimSpace(string(p))

	if _, exists := h.m[name]; exists {
		return fmt.Errorf("hub: duplicate auth name(%s)", name)
	}

	c.name = name
	h.m[name] = c
	auth, err := h.store.GetLast(name)
	if err != nil {
		return err
	}
	// 取得上次離線時間
	c.SetLastAuth(auth)
	h.Printf("[hub] %s last: %+v\n", name, auth)
	nowAuth := c.GetAuth()
	h.Printf("[hub] %s auth: %+v\n", name, nowAuth)

	return h.store.NewAuth(nowAuth)
}

// Quit 執行退出紀錄
func (h *Hub) quit(conn *Conn, t time.Time) {
	h.Lock()
	defer h.Unlock()
	defer conn.conn.Close()

	delete(h.m, conn.name)
	auth := conn.GetAuth()
	auth.DisconnectedAt = t.Unix()
	if err := h.store.UpdateAuth(auth); err != nil {
		h.Println(err)
	}
	h.Printf("[hub] %s quit: %+v\n", conn.name, auth)
}

func (h *Hub) quitAll(t time.Time) {
	m := map[string]*Conn{}
	h.RLock()
	for n, c := range h.m {
		m[n] = c
	}
	h.RUnlock()

	for _, c := range m {
		h.quit(c, t)
	}
}

func (h *Hub) publish(e *store.Event) int {
	h.RLock()
	defer h.RUnlock()

	var cnt int
	for name, conn := range h.m {
		if conn.isListening(e.Name) {
			cnt++
			h.Printf("write to %s: %s\n", name, e.Raw)
			go conn.writeEvent(e.Raw)
		}
	}

	return cnt
}

func (h *Hub) recover(conn *Conn, since int64) error {
	h.Printf("recover: %+v since %d\n", conn, since)

	if since == 0 {
		since = conn.lastAuth.DisconnectedAt
	}

	h.store.EachEvents(func(e *store.Event) {
		if conn.isListening(e.Name) {
			h.Printf("resend %s: %+v\n", conn.name, e)
			conn.writeEvent(e.Raw)
		}
	}, since, nil)

	return nil
}

func (h *Hub) handle(c *Conn) {

	defer func() {
		c.flush()
		h.quit(c, time.Now())
		if c.err != nil {
			h.Println("conn error:", c.err)
		}
	}()

	for {
		line, err := c.readLine()
		log.Printf("<- client [%s]: %v = [%s] %v\n", c.conn.RemoteAddr().String(), line, line, err)
		if err != nil {
			return
		}

		/*
			[server]
			$xxx // 登入名稱
			=n = len(event:{...}) // 事件長度(int)
			event:{...} // 事件本體
			+channal // 註冊頻道
			-channal // 移除頻道

			[client]
			!xxx // 錯誤訊息
			=n // 事件長度(int)
			event:{...} // 事件本體
			*OK // 請求處理完成
		*/
		switch line[0] {
		case CPing:
			c.writePong()
		case CAuth:
			// 失敗直接斷線
			if err := h.auth(c, line[1:]); err != nil {
				log.Println(err)
				c.writeError(err)
				return
			}

		case CRecover:
			n, err := parseLen(line[1:])
			if err != nil {
				log.Println(err)
				c.writeError(err)
				return
			}
			h.recover(c, n)

		case CAddChan:
			if channal, err := c.subscribe(line[1:]); err != nil {
				log.Printf("%s subscribe failed %v\n", c.name, err)
				c.writeError(err)
			} else {
				log.Printf("%s subscribe %s\n", c.name, channal)
				c.writeOk()
			}

		case CDelChan:
			log.Printf("%s unsubscribe %s\n", c.name, line[1:])
			if err := c.unsubscribe(line[1:]); err != nil {
				log.Printf("%s unsubscribe failed %v\n", c.name, err)
				c.writeError(err)
			} else {
				c.writeOk()
			}

		case CLength:
			n, err := parseLen(line[1:])
			if err != nil {
				log.Println(err)
				c.writeError(err)
				return
			}

			log.Printf("length: %d\n", n)

			p := make([]byte, n)
			log.Printf("read: %+v = [%s]\n", p, p)
			_, err = io.ReadFull(c.r, p)
			// 去除換行
			c.readLine()
			if err != nil {
				log.Println(err)
				c.writeError(err)
				return
			}

			eventName, eventData, err := parseEvent(p)
			log.Println("receive:", string(eventName), string(eventData), err)
			if err == nil {
				storeEvent := makeEvent(eventName, eventData, time.Now())
				go func() {
					// 排隊寫入
					h.store.Events <- storeEvent
				}()
				h.publish(storeEvent)
			} else {
				c.writeError(err)
			}
		}
	}
}

// ListenAndServe listen address and serve conn
func (h *Hub) ListenAndServe(quit <-chan os.Signal, addr string) error {

	network := "tcp"

	tcpAddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(network, tcpAddr)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("conn: ", err)
				return
			}
			go h.handle(newConn(conn, time.Now()))
		}
	}()

	s := <-quit
	log.Printf("Receive os.Signal %s\n", s)
	h.quitAll(time.Now())
	return listener.Close()

}

func makeEventStream(event, data []byte) []byte {
	buf := bytes.NewBuffer(event)
	buf.WriteByte(':')
	buf.Write(data)

	return buf.Bytes()
}

func makeEvent(ev, data []byte, t time.Time) *store.Event {
	p := makeEventStream(ev, data)
	e := event.Event(string(ev))
	return &store.Event{
		Hash:       fmt.Sprintf("%x", sha1.Sum(p)),
		Name:       e.String(),
		Prefix:     e.Type(),
		Length:     len(p),
		Raw:        string(p),
		ReceivedAt: t.Unix(),
	}
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

func parseChannal(p []byte) (channal string, err error) {

	// 去除空白/換行
	channal = strings.TrimSpace(string(p))
	if channal == "" {
		err = errors.New("empty channal")
	}

	return
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
