package listener

import (
	"errors"
	"log"
	"sync"

	"github.com/colindev/events/event"
	"github.com/colindev/events/redis"
)

type (
	// Listener responsible for trigger match handlers
	Listener interface {
		On(event.Event, ...event.Handler) Listener
		Trigger(event.Event, event.RawData)
		TriggerRecover(func(interface{}))
		Run(channels ...interface{}) error
		Stop() error
		Ping(string) error
	}

	listener struct {
		*sync.RWMutex
		pool           redis.Pool
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

// New return Listener instansce
func New(pool redis.Pool) Listener {
	return &listener{
		RWMutex: &sync.RWMutex{},
		pool:    pool,
		events:  make(map[event.Event][]event.Handler),
	}
}

func (l *listener) On(ev event.Event, hs ...event.Handler) Listener {
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
	for _, fn := range hs {
		go fn(ev, rd)
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

func (l *listener) Stop() error {
	l.RLock()
	defer l.RUnlock()
	if l.running {
		return l.psc.PUnsubscribe()
	}

	return nil
}
