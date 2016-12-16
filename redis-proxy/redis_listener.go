package main

import (
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/colindev/events/event"
	eventsListener "github.com/colindev/events/listener"
	"github.com/garyburd/redigo/redis"
)

type (
	listener struct {
		wg sync.WaitGroup
		sync.RWMutex
		pool           *redis.Pool
		psc            redis.PubSubConn
		running        bool
		events         map[event.Event][]event.Handler
		triggerRecover func(interface{})
	}
)

var (
	// ErrListenerAlreadyRunning prevent run second time
	ErrListenerAlreadyRunning = errors.New("[events] Listener already running")
	// ErrListenerNotRunning prevent access nil resource
	ErrListenerNotRunning = errors.New("[events] Listener not running")
)

// NewRedisListener return Listener instansce
func NewRedisListener(pool *redis.Pool) eventsListener.Listener {
	return &listener{
		pool:   pool,
		events: make(map[event.Event][]event.Handler),
	}
}

func (l *listener) Recover(int64, int64) error {
	return errors.New("not support")
}

func (l *listener) On(ev event.Event, hs ...event.Handler) eventsListener.Listener {
	l.Lock()
	defer l.Unlock()

	if _, ok := l.events[ev]; !ok {
		l.events[ev] = []event.Handler{}
	}
	l.events[ev] = append(l.events[ev], hs...)

	return l
}

func (l *listener) Run(channels ...interface{}) (err error) {

	err = func() error {
		l.Lock()
		defer l.Unlock()
		if l.running {
			return ErrListenerAlreadyRunning
		}
		l.running = true
		return nil
	}()
	if err != nil {
		return
	}

	// 斷線的話就重新設定 running = false
	defer func() {
		l.running = false
	}()

	conn := l.pool.Get()
	defer conn.Close()
	l.psc.Conn = conn
	l.psc.PSubscribe(channels...)

	for {
		m := l.psc.Receive()
		switch m := m.(type) {
		case redis.Message:
			go l.Trigger(event.Event(m.Channel), event.RawData(m.Data))
		case redis.PMessage:
			go l.Trigger(event.Event(m.Channel), event.RawData(m.Data))
		case redis.Pong:
			go l.Trigger(event.PONG, event.RawData(m.Data))
		case redis.Subscription:
			log.Println("[events]", m)
		case error:
			return m
		}
	}
}

func (l *listener) WaitHandler() error {
	l.RLock()
	running := l.running
	l.RUnlock()
	if running {
		return l.psc.PUnsubscribe()
	}

	l.wg.Wait()

	return nil
}

func (l *listener) RunForever(quit chan os.Signal, reconn time.Duration, chs ...interface{}) eventsListener.Listener {

	go func() {
		s := <-quit
		l.RLock()
		psc := l.psc
		l.RUnlock()

		psc.Close()

		quit <- s

	}()

	for {
		select {
		case <-quit:
			return l
		default:
			l.Run(chs...)
			time.Sleep(reconn)
		}
	}
}

func (l *listener) Trigger(ev event.Event, rd event.RawData) {

	l.RLock()
	tr := l.triggerRecover
	l.RUnlock()
	if tr != nil {
		defer func() {
			if r := recover(); r != nil {
				tr(r)
			}
		}()
	}

	hs := l.findHandlers(ev)
	for _, handler := range hs {
		l.wg.Add(1)
		go func(fn event.Handler) {
			fn(ev, rd)
			l.wg.Done()
		}(handler)
	}
}

func (l *listener) TriggerRecover(tr func(interface{})) {
	l.Lock()
	defer l.Unlock()
	l.triggerRecover = tr
}

func (l *listener) findHandlers(target event.Event) []event.Handler {
	ret := []event.Handler{}
	l.RLock()
	defer l.RUnlock()

	for ev, hs := range l.events {
		if ev.Match(target) {
			ret = append(ret, hs...)
		}
	}

	return ret
}

func (l *listener) Ping(msg string) error {
	if l.psc.Conn == nil {
		return ErrListenerNotRunning
	}

	return l.psc.Ping(msg)
}
