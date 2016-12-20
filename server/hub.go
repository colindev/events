package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/store"
)

// Hub 負責管理連線
type Hub struct {
	// 連線鎖
	sync.RWMutex
	// 需作歷程管理
	m map[string]Conn
	// 不須作歷程管理
	g map[Conn]bool
	// 等待連線全部退出用
	sync.WaitGroup

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
		m:      map[string]Conn{},
		g:      map[Conn]bool{},
		store:  sto,
		Logger: logger,
	}, nil
}

// Auth 執行登入紀錄
func (h *Hub) auth(c Conn, p []byte) error {

	// log.Printf("\033[31mauth %s(%s)\033[m\n", c.RemoteAddr(), c.GetName())

	h.Lock()
	defer h.Unlock()

	// 設定讀寫權限
	s := strings.SplitN(strings.TrimSpace(string(p)), ":", 2)
	if len(s) != 2 {
		return fmt.Errorf("auth data schema error: %s", p)
	}
	name := s[0]
	flags, err := strconv.Atoi(s[1])
	if err != nil {
		return fmt.Errorf("auth flags error: %v", err)
	}
	c.SetFlags(flags)
	h.Println("auth[name]=", name)
	h.Println("auth[flags]=", client.Flag(flags).String())

	// 匿名登入
	if name == "" {
		h.g[c] = true
		return nil
	}

	if c, exists := h.m[name]; exists {
		return fmt.Errorf("hub: duplicate auth %s(%s)", c.RemoteAddr(), name)
	}

	c.SetName(name)
	if c.IsAuthed() {
		for n, x := range h.m {
			if x == c {
				delete(h.m, n)
				break
			}
		}
	}
	h.m[name] = c
	c.SetAuthed(true)
	auth, err := h.store.GetLast(name)
	if err != nil {
		return err
	}
	// 取得上次離線時間
	c.SetLastAuth(auth)
	h.Printf("[hub] %s last: %+v\n", name, auth)
	nowAuth := c.GetAuth()
	h.Printf("[hub] %s auth: %v\n", name, nowAuth)
	return h.store.NewAuth(nowAuth)
}

// Quit 執行退出紀錄
func (h *Hub) quit(c Conn, t time.Time) (auth *store.Auth) {
	var err error
	h.Lock()
	defer func() {
		h.Unlock()
		c.Close(err)
	}()
	auth = c.GetAuth()
	auth.DisconnectedAt = t.Unix()

	if !c.HasName() {
		delete(h.g, c)
		return
	}

	delete(h.m, c.GetName())
	if err := h.store.UpdateAuth(auth); err != nil {
		h.Println(err)
	}
	h.Printf("[hub] %s quit: %+v\n", c.GetName(), auth)
	return
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
	var conns = []Conn{}
	for _, c := range h.m {
		if c.IsListening(e.Name) {
			conns = append(conns, c)
		}
	}

	for c := range h.g {
		if c.IsListening(e.Name) {
			conns = append(conns, c)
		}
	}
	h.RUnlock()

	h.Println("publish: ", e.Raw)
	var cnt int
	for _, c := range conns {
		cnt++
		h.Printf("send to: %s(%s)", c.RemoteAddr(), c.GetName())
		go c.SendEvent(e.Raw)
	}

	return cnt
}

func (h *Hub) recover(c Conn, since, until int64) error {

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

	if until <= 0 {
		until = time.Now().Unix()
	}

	prefix := []string{}
	chs := c.EachChannels(func(ch string, re *regexp.Regexp) string {
		prefix = append(prefix, event.Event(ch).Type())
		return ch
	})

	h.Printf("recover: %s(%s) since=%d until=%d channels=%v\n", c.RemoteAddr(), c.GetName(), since, until, chs)

	err := h.store.EachEvents(func(e *store.Event) error {
		if c.IsListening(e.Name) {
			// 先不浪費I/O了
			// h.Printf("resend %s: %+v\n", c.GetName(), e)
			return c.SendEvent(e.Raw)
		}
		return nil
	}, prefix, since, until)

	if err != nil {
		return err
	}

	auth := c.GetAuth()
	auth.RecoverSince = since
	auth.RecoverUntil = until

	return h.store.UpdateAuth(auth)
}

func (h *Hub) handle(c Conn) {

	h.Add(1)
	defer func() {
		h.quit(c, time.Now())
		h.Done()
		if err := c.Err(); err != nil {
			h.Println("conn error:", err)
		}
		h.Println(c.RemoteAddr(), " disconnect")
	}()

	// 登入的第一個訊息一定是登入訊息
	line, err := c.ReadLine()
	if err != nil {
		return
	} else if err := h.auth(c, line[1:]); err != nil {
		h.Println(err)
		c.SendError(err)
		return
	}

	c.SendEvent(fmt.Sprintf("%s:ok", event.Connected))

	h.Println(c.RemoteAddr(), " connect")
	for {
		line, err := c.ReadLine()
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
				since, until int64
			)
			// TODO 先可以跑
			// 後面改 byte 處理
			s := strings.SplitN(string(line[1:]), ":", 2)
			since, _ = strconv.ParseInt(s[0], 10, 64)
			if len(s) == 2 {
				until, _ = strconv.ParseInt(s[1], 10, 64)
			}

			if err := h.recover(c, since, until); err != nil {
				h.Printf("recover since=%d until=%d error: %v\n", since, until, err)
				c.SendError(err)
			}

		case client.CAddChan:
			if channel, err := c.Subscribe(line[1:]); err != nil {
				h.Printf("app(%s) subscribe failed %v\n", c.GetName(), err)
				c.SendError(err)
			} else {
				h.Printf("app(%s) subscribe [%s]\n", c.GetName(), channel)
				c.SendReply("subscribe " + channel + " OK")
			}

		case client.CDelChan:
			if channel, err := c.Unsubscribe(line[1:]); err != nil {
				h.Printf("app(%s) unsubscribe failed %v\n", c.GetName(), err)
				c.SendError(err)
			} else {
				h.Printf("app(%s) unsubscribe [%s]\n", c.GetName(), channel)
				c.SendReply("unsubscribe " + channel + " OK")
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

			if !c.Writable() {
				h.Printf("this (%p)%#v has no writable flag, event droped\n", c.(*conn), c)
				continue
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
func (h *Hub) ListenAndServe(quit <-chan os.Signal, addr string, others ...Conn) error {

	var err error
	network := "tcp"

	tcpAddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(network, tcpAddr)
	if err != nil {
		return err
	}

	for _, c := range others {
		go h.handle(c)
	}
	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				h.Println("conn: ", err)
				return
			}
			go h.handle(newConn(c, time.Now()))
		}
	}()

	s := <-quit
	h.Printf("Receive os.Signal %s\n", s)
	h.quitAll(time.Now())
	h.Println("h.quitAll")
	err = listener.Close()
	h.Println("close listener")
	h.Wait()
	h.store.Close()
	h.Println("close store")

	return err
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

func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func isBetween(a, b, c int64) bool {
	n, m := min(b, c), max(b, c)

	return a >= n && a <= m
}
