package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/colindev/events/server/store"
)

// Hub 負責管理連線
type Hub struct {
	*sync.RWMutex
	m     map[string]*Conn
	store *store.Store
	*log.Logger
}

func NewHub(env *Env, logger *log.Logger) (*Hub, error) {

	sto, err := store.New(store.Config{
		Debug:    env.Debug,
		AuthDSN:  env.AuthDSN,
		EventDSN: env.EventDSN,
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
func (h *Hub) Quit(conn *Conn, t time.Time) {
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

func (h *Hub) Handler(c *Conn) {

	defer c.close()
	defer c.w.Flush()
	defer func() {
		h.Quit(c, time.Now())
	}()

	log.Printf("%#v\n", c.conn.RemoteAddr().String())

	for {
		line, err := c.readLine()
		log.Printf("<- client: %v = %s %v\n", line, line, err)
		if err != nil {
			return
		}

		/*
			$xxx = auth
			:n = len(event:{...})
			event:{...}
			+channal
		*/
		switch line[0] {
		case '$':
			// 失敗直接斷線
			if err := h.auth(c, line[1:]); err != nil {
				log.Println(err)
				c.w.Write([]byte("!" + err.Error()))
				return
			}
		case '+':
			c.subscribe(line[1:])
		case ':':
			n, err := parseLen(line[1:])
			if err != nil {
				log.Println(err)
				c.w.Write([]byte("!" + err.Error()))
				return
			}

			log.Printf("length: %d\n", n)

			p := make([]byte, n)
			_, err = io.ReadFull(c.r, p)
			if err != nil {
				log.Println(err)
				c.w.Write([]byte("!" + err.Error()))
				return
			}

			// TODO 存起來
			eventName, eventData := parseEvent(p)
			log.Println("receive:", eventName, string(eventData))
			// 去除換行
			c.readLine()

		}
		if err := c.w.Flush(); err != nil {
			log.Println(err)
		}

	}
}
