package listener

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
)

type (
	// Listener responsible for trigger match handlers
	Listener interface {
		On(event.Event, ...event.Handler) Listener
		Recover(int64, int64) error
		Trigger(event.Event, event.RawData)
		TriggerRecover(func(interface{}))
		Run(channels ...interface{}) error
		RunForever(chan os.Signal, time.Duration, ...interface{}) Listener
		WaitHandler() error
		Ping(string) error
	}

	listener struct {
		wg *sync.WaitGroup
		*sync.RWMutex
		conn           client.Conn
		chs            []string
		dial           func() (client.Conn, error)
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
func New(dial func() (client.Conn, error)) Listener {
	return &listener{
		wg:      &sync.WaitGroup{},
		RWMutex: &sync.RWMutex{},
		dial:    dial,
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

	// 斷線的話就重新設定 running = false
	defer func() {
		l.running = false
	}()

	// 建立連線
	conn, err := l.dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	l.conn = conn

	defer func() {
		var rd event.RawData
		if err != nil {
			rd = event.RawData(err.Error())
		}
		l.Trigger(event.Disconnected, rd)
	}()
	// 登入名稱
	if err := conn.Auth(client.Readable); err != nil {
		return err
	}

	// 轉換頻道
	convChans := []string{}
	for _, c := range channels {
		switch v := c.(type) {
		case string:
			convChans = append(convChans, v)
		case event.Event:
			convChans = append(convChans, string(v))
		case []byte:
			convChans = append(convChans, string(v))
		default:
			convChans = append(convChans, fmt.Sprintf("%v", v))
		}
	}

	if err := conn.Subscribe(convChans...); err != nil {
		return err
	}
	l.chs = convChans

	padding := len(l.chs)
	for {
		var m interface{}
		m, err = conn.Receive()
		if err != nil {
			return err
		}

		switch m := m.(type) {
		case *client.Event:
			go l.Trigger(m.Name, m.Data)
		case *client.Reply:
			if padding > 0 {
				padding--
				if padding == 0 {
					go l.Trigger(event.Ready, nil)
				}
			}
			log.Println("[event] reply ", m.String())
		default:
			log.Printf("[event] unexpect %#v\n", m)
		}
	}
}

func (l *listener) WaitHandler() error {
	l.RLock()
	defer l.RUnlock()
	if l.running {
		return l.conn.Unsubscribe(l.chs...)
	}

	l.wg.Wait()

	return nil
}

func (l *listener) RunForever(quit chan os.Signal, reconnDuration time.Duration, chs ...interface{}) Listener {
	for {
		select {
		case <-quit:
			return l
		default:
			l.Run(chs...)
			time.Sleep(reconnDuration)
		}
	}
}

// Recover 包裝 conn Recover / RecoverSince
// i 小於等於 0 時, 調用 conn.Recover() 讓 server 處理還原時間點
// i 大於 0 時, 調用 onn.RecoverSince(timestamp) 由 client 決定還原時間點
func (l *listener) Recover(since, until int64) error {
	return l.conn.Recover(since, until)
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
	if l.conn == nil {
		return ErrListenerNotRunning
	}

	return l.conn.Ping(msg)
}
