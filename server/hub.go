package main

import (
	"errors"
	"fmt"
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
func (h *Hub) Auth(c *Conn, p []byte) error {

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
	h.Printf("[hub] %s last: %+v\n", name, auth)
	h.Printf("[hub] %s auth: %+v\n", name, c.GetAuth())

	// 取得上次離線時間
	c.SetLastAuth(auth)
	return nil
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
