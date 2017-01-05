package main

import (
	"fmt"
	"log"
	"net"
	"os"
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

	// log verbose
	verbose bool
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
		m:       map[string]Conn{},
		g:       map[Conn]bool{},
		store:   sto,
		Logger:  logger,
		verbose: env.Debug,
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

// 為了方便從資料庫取出廣播,不要改變第一參數型別
func (h *Hub) publish(e *store.Event, ignore ...Conn) int {

	h.RLock()
	var conns = []Conn{}
	var ignores = map[Conn]bool{}

	// 先整理要忽略的連線
	for _, c := range ignore {
		ignores[c] = true
	}

	for _, c := range h.m {
		if c.IsListening(e.Name) && !ignores[c] {
			conns = append(conns, c)
		}
	}

	for c := range h.g {
		if c.IsListening(e.Name) && !ignores[c] {
			conns = append(conns, c)
		}
	}
	h.RUnlock()

	// broadcast
	var cnt int
	for _, c := range conns {
		cnt++
		h.Printf("send to: %s(%s)", c.RemoteAddr(), c.GetName())
		go c.SendEvent(e.Raw)
	}

	return cnt
}

func (h *Hub) sendEventTo(app string, e *store.Event) error {
	h.RLock()
	c := h.m[app]
	h.RUnlock()

	if c != nil && c.IsListening(e.Name) {
		return c.SendEvent(e.Raw)
	}

	return nil
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
	hasMatchAll := false
	chs := c.EachChannels(func(ch event.Event) event.Event {
		group := ch.Type()
		if group == "*" {
			hasMatchAll = true
		}
		prefix = append(prefix, group)
		return ch
	})
	if hasMatchAll {
		prefix = []string{}
	}

	h.Printf("recover: %s(%s) since=%d until=%d channels=%v\n", c.RemoteAddr(), c.GetName(), since, until, chs)

	err := h.store.EachEvents(func(e *store.Event) error {
		if c.IsListening(e.Name) {
			// 先不浪費I/O了
			// h.Printf("resend %s: %+v\n", c.GetName(), e)
			return c.SendEvent(e.Raw)
		}
		return nil
	}, prefix, since, until)

	if err == nil && c.GetName() != "" {
		auth := c.GetAuth()
		auth.RecoverSince = since
		auth.RecoverUntil = until
		return h.store.UpdateAuth(auth)
	}

	return err
}

func (h *Hub) handle(c Conn) {

	h.Add(1)
	defer func() {
		h.publishQuit(c, h.quit(c, time.Now()))
		h.Done()
		if err := c.Err(); err != nil {
			h.Println("conn error:", err)
		}
		h.Println(c.RemoteAddr(), " disconnect")
	}()

	if !c.IsAuthed() {
		// 登入的第一個訊息一定是登入訊息
		line, err := c.ReadLine()
		if err != nil {
			return
		} else if err := h.auth(c, line[1:]); err != nil {
			h.Println(err)
			c.SendError(err)
			return
		}
	}

	h.Println(c.RemoteAddr(), " connect")

	// 發送連線事件
	c.SendEvent(string(client.MakeEventStream(event.Connected, client.OK)))

	// 廣播具名客端 Join 事件
	h.publishJoin(c)

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
		// case client.CAuth: 只做單次登入
		case client.CRecover:
			since, until := client.ParseSinceUntil(line[1:])

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

		case client.CInfo:
			rd, err := event.Compress(event.RawData(h.info(true)))
			if err != nil {
				c.SendError(err)
			} else {
				c.SendEvent(string(client.MakeEventStream(event.Info, rd)))
			}

		case client.CTarget:
			target, p, err := c.ReadTargetAndLen(line[1:])
			if err != nil {
				h.Println(err)
				c.SendError(err)
				return
			}

			if !c.Writable() {
				h.Printf("this (%p)%#v has no writable flag, event droped\n", c.(*conn), c)
				if h.verbose {
					h.Printf("\n--- payload\n%s\n---", p)
				}
				continue
			}

			eventName, eventData, err := client.ParseEvent(p)
			s, _ := event.Uncompress(eventData)
			h.Printf("from %s to %s: %s %s %v", c.RemoteAddr(), target, eventName, s, err)
			// 指定傳送不儲存
			if err == nil {
				storeEvent := client.MakeEvent(eventName, eventData, time.Now())
				h.sendEventTo(target, storeEvent)
			} else {
				c.SendError(err)
			}

		case client.CEvent:
			p, err := c.ReadLen(line[1:])
			if err != nil {
				h.Println(err)
				c.SendError(err)
				return
			}

			if !c.Writable() {
				h.Printf("this (%p)%#v has no writable flag, event droped\n", c.(*conn), c)
				if h.verbose {
					eventName, eventData, err := client.ParseEvent(p)
					s, _ := event.Uncompress(eventData)
					h.Printf("\n--- payload\n%s %s\n%v\n---", eventName, s, err)
				}
				continue
			}

			eventName, eventData, err := client.ParseEvent(p)
			s, _ := event.Uncompress(eventData)
			h.Printf("from %s broadcast: %s %s %v", c.RemoteAddr(), eventName, s, err)
			if err == nil {
				storeEvent := client.MakeEvent(eventName, eventData, time.Now())
				h.store.Events <- storeEvent
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

func (h *Hub) publishJoin(c Conn) error {

	if !c.HasName() {
		return nil
	}
	connAuth := c.GetAuth()
	rd, err := event.Marshal(connAuth)
	if err != nil {
		return err
	}
	rdCompressed, err := event.Compress(rd)
	if err != nil {
		return err
	}
	storeEvent := client.MakeEvent(event.Join, rdCompressed, time.Unix(connAuth.ConnectedAt, 0))
	h.publish(storeEvent, c)

	return nil
}

func (h *Hub) publishQuit(c Conn, auth *store.Auth) error {
	if !c.HasName() {
		return nil
	}
	rd, err := event.Marshal(auth)
	if err != nil {
		return err
	}
	rdCompressed, err := event.Compress(rd)
	if err != nil {
		return err
	}
	storeEvent := client.MakeEvent(event.Leave, rdCompressed, time.Unix(auth.DisconnectedAt, 0))
	h.publish(storeEvent, c)

	return nil
}

func (h *Hub) info(ignoreWriteOnly bool) string {

	h.RLock()
	defer h.RUnlock()

	ghost := []*ConnStatus{}
	for c := range h.g {
		if st := c.(*conn).Status(ignoreWriteOnly); st != nil {
			ghost = append(ghost, st)
		}
	}

	auth := map[string]*ConnStatus{}
	for name, c := range h.m {
		if st := c.(*conn).Status(ignoreWriteOnly); st != nil {
			auth[name] = st
		}
	}

	b, _ := event.Marshal(struct {
		Auth  map[string]*ConnStatus
		Ghost []*ConnStatus
	}{auth, ghost})

	return string(b)
}
