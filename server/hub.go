package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/server/store"
)

// Hub 負責管理連線
type Hub struct {
	*sync.RWMutex
	// 需作歷程管理
	m map[string]Conn
	// 不須作歷程管理
	g     map[Conn]bool
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
		m:       map[string]Conn{},
		g:       map[Conn]bool{},
		store:   sto,
		Logger:  logger,
	}, nil
}

// Auth 執行登入紀錄
func (h *Hub) auth(c Conn, p []byte) error {

	h.Lock()
	defer h.Unlock()

	name := strings.TrimSpace(string(p))

	// 匿名登入
	if name == "" {
		h.g[c] = true
		return nil
	}

	if _, exists := h.m[name]; exists {
		return fmt.Errorf("hub: duplicate auth name(%s)", name)
	}

	c.SetName(name)
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
func (h *Hub) quit(c Conn, t time.Time) {
	var err error
	h.Lock()
	defer h.Unlock()
	defer c.Close(err)

	if !c.HasName() {
		delete(h.g, c)
		return
	}

	delete(h.m, c.GetName())
	auth := c.GetAuth()
	auth.DisconnectedAt = t.Unix()
	if err := h.store.UpdateAuth(auth); err != nil {
		h.Println(err)
	}
	h.Printf("[hub] %s quit: %+v\n", c.GetName(), auth)
}

func (h *Hub) quitAll(t time.Time) {
	m := map[Conn]bool{}
	h.RLock()
	for _, c := range h.m {
		m[c] = true
	}
	for c := range h.g {
		m[c] = true
	}
	h.RUnlock()

	for c := range m {
		h.quit(c, t)
	}
}

func (h *Hub) publish(e *store.Event) int {
	h.RLock()
	defer h.RUnlock()

	var cnt int
	for name, c := range h.m {
		if c.IsListening(e.Name) {
			cnt++
			h.Printf("write to %s: %s\n", name, e.Raw)
			go c.SendEvent(e.Raw)
		}
	}

	for c := range h.g {
		if c.IsListening(e.Name) {
			cnt++
			h.Printf("write to ghost: %s\n", e.Raw)
			go c.SendEvent(e.Raw)
		}
	}

	return cnt
}

func (h *Hub) recover(c Conn, since int64) error {
	h.Printf("recover: %+v since %d\n", c, since)

	if since == 0 {
		// NOTE 沒上一次的登入紀錄時不重送全部訊息
		lastAuth := c.GetLastAuth()
		if lastAuth == nil {
			return nil
		}
		if lastAuth.DisconnectedAt == 0 {
			return nil
		}
		since = lastAuth.DisconnectedAt
	}

	prefix := []string{}
	c.EachChannels(func(ch string, re *regexp.Regexp) string {
		prefix = append(prefix, event.Event(ch).Type())
		return ""
	})

	h.store.EachEvents(func(e *store.Event) {
		if c.IsListening(e.Name) {
			h.Printf("resend %s: %+v\n", c.GetName(), e)
			c.SendEvent(e.Raw)
		}
	}, since, prefix)

	return nil
}

func (h *Hub) handle(c Conn) {

	defer func() {
		h.quit(c, time.Now())
		if err := c.Err(); err != nil {
			h.Println("conn error:", err)
		}
	}()

	for {
		line, err := c.ReadLine()
		h.Printf("<- client [%s]: %v = [%s] %v\n", c.RemoteAddr(), line, line, err)
		if err != nil {
			return
		}

		if line == nil {
			// drop empty line
			continue
		}

		switch line[0] {
		case client.CAuth:
			// 失敗直接斷線
			if err := h.auth(c, line[1:]); err != nil {
				h.Println(err)
				c.SendError(err)
				return
			}

		case client.CRecover:
			var (
				n   int64
				err error
			)
			if len(line[1:]) > 0 {
				n, err = parseLen(line[1:])
			}
			if err != nil {
				h.Println(err)
				c.SendError(err)
				return
			}
			h.recover(c, n)

		case client.CAddChan:
			if channel, err := c.Subscribe(line[1:]); err != nil {
				h.Printf("app(%s) subscribe failed %v\n", c.GetName(), err)
				c.SendError(err)
			} else {
				h.Printf("app(%s) subscribe [%s]\n", c.GetName(), channel)
				c.SendReply("OK")
			}

		case client.CDelChan:
			if channel, err := c.Unsubscribe(line[1:]); err != nil {
				h.Printf("app(%s) unsubscribe failed %v\n", c.GetName(), err)
				c.SendError(err)
			} else {
				h.Printf("app(%s) unsubscribe [%s]\n", c.GetName(), channel)
				c.SendReply("OK")
			}

		case client.CPing:
			p, err := c.ReadLen(line[1:])
			if err != nil {
				h.Println(err)
				c.SendError(err)
				return
			}

			c.SendPong(p)

		case client.CEvent:
			p, err := c.ReadLen(line[1:])
			if err != nil {
				h.Println(err)
				c.SendError(err)
				return
			}

			eventName, eventData, err := parseEvent(p)
			h.Println("receive:", string(eventName), string(eventData), err)
			if err == nil {
				storeEvent := makeEvent(eventName, eventData, time.Now())
				go func() {
					// 排隊寫入
					h.store.Events <- storeEvent
				}()
				h.publish(storeEvent)
			} else {
				c.SendError(err)
			}
		default:
			h.Println("droped")
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
				h.Println("conn: ", err)
				return
			}
			go h.handle(newConn(conn, time.Now()))
		}
	}()

	s := <-quit
	h.Printf("Receive os.Signal %s\n", s)
	h.quitAll(time.Now())
	h.Println("h.quitAll")
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
