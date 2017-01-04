package launcher

import (
	"container/list"
	"sync"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
)

type (
	// Launcher responsible for event data
	Launcher interface {
		Fire(event.Event, event.RawData) error
		FireTo(string, event.Event, event.RawData) error
		Close() error
	}

	launcher struct {
		wg   sync.WaitGroup
		pool client.Pool
		c    chan *client.Event
	}
)

// New return a Launcher instance
func New(pool client.Pool) Launcher {
	l := &launcher{
		pool: pool,
		c:    make(chan *client.Event, 100),
	}

	l.wg.Add(1)
	go l.reduce(5, time.Millisecond*300)

	return l
}

func (l *launcher) Fire(ev event.Event, rd event.RawData) error {

	l.c <- &client.Event{
		Name: ev,
		Data: rd,
	}

	return nil
}

func (l *launcher) FireTo(target string, ev event.Event, rd event.RawData) error {
	l.c <- &client.Event{
		Target: target,
		Name:   ev,
		Data:   rd,
	}

	return nil
}

// Run try resent
func (l *launcher) reduce(keep int, du time.Duration) {
	conn := l.pool.Get()
	conn.Auth(client.Writable)
	// 保留最後 n 筆資料
	cache := list.New()
	fire := func(c client.Conn, ca *client.Event) error {
		var err error
		switch ca.Target {
		case "":
			err = c.Fire(ca.Name, ca.Data)
		default:
			err = c.FireTo(ca.Target, ca.Name, ca.Data)
		}
		return err
	}
	for ca := range l.c {
		for cache.Len() > keep {
			cache.Remove(cache.Front())
		}
		for {
			if err := fire(conn, ca); err == nil {
				cache.PushBack(ca)
				break
			}
			conn.Close()
			conn = l.pool.Get()
			if err := conn.Auth(client.Writable); err != nil {
				time.Sleep(du)
			} else {
				// 每次斷線後重新連上,延遲一段時間才重送資料
				time.Sleep(time.Second * 10)

				// 重送斷線前 n 筆資料
				el := cache.Front()
				for el != nil {
					fire(conn, el.Value.(*client.Event))
					el = el.Next()
				}
			}
		}
	}

	l.wg.Done()
}

func (l *launcher) Close() error {
	close(l.c)
	l.wg.Wait()
	return nil
}
