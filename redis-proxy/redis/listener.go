package redis

import (
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/colindev/events/event"
	eventsListener "github.com/colindev/events/listener"
	x "github.com/garyburd/redigo/redis"
)

type (
	Listener struct {
		wg sync.WaitGroup
		sync.RWMutex
		pool           *x.Pool
		psc            x.PubSubConn
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

// NewListener return Listener instansce
func NewListener(pool *Pool) eventsListener.Listener {
	return &Listener{
		pool:   pool.Pool,
		events: make(map[event.Event][]event.Handler),
	}
}

func (l *Listener) Recover(int64, int64) error {
	return errors.New("not support")
}

func (l *Listener) On(ev event.Event, hs ...event.Handler) eventsListener.Listener {
	l.Lock()
	defer l.Unlock()

	if _, ok := l.events[ev]; !ok {
		l.events[ev] = []event.Handler{}
	}
	l.events[ev] = append(l.events[ev], hs...)

	return l
}

func (l *Listener) Run(channels ...interface{}) (err error) {

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
	err = l.psc.PSubscribe(channels...)
	if err != nil {
		return
	}

	for {
		m := l.psc.Receive()
		switch m := m.(type) {
		case x.Message:
			go l.Trigger(event.Event(m.Channel), event.RawData(m.Data))
		case x.PMessage:
			go l.Trigger(event.Event(m.Channel), event.RawData(m.Data))
		case x.Pong:
			go l.Trigger(event.PONG, event.RawData(m.Data))
		case x.Subscription:
			log.Println("[events]", m)
		case error:
			return m
		}
	}
}

func (l *Listener) WaitHandler() error {

	l.wg.Wait()

	return nil
}

func (l *Listener) RunForever(quit chan os.Signal, reconn time.Duration, chs ...interface{}) eventsListener.Listener {

	go func() {
		for {
			l.Run(chs...)
			time.Sleep(reconn)
		}
	}()

	<-quit
	return l
}

func (l *Listener) Trigger(ev event.Event, rd event.RawData) {

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

func (l *Listener) TriggerRecover(tr func(interface{})) {
	l.Lock()
	defer l.Unlock()
	l.triggerRecover = tr
}

func (l *Listener) findHandlers(target event.Event) []event.Handler {
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

func (l *Listener) Ping(msg string) error {
	if l.psc.Conn == nil {
		return ErrListenerNotRunning
	}

	return l.psc.Ping(msg)
}

// 不實作
func (l *Listener) Info() error { return nil }
