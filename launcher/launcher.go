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

func (l *launcher) Fire(ev event.Event, rd event.RawData) (err error) {

	data, err := event.Compress(rd)
	if err != nil {
		return
	}
	l.c <- &client.Event{
		Name: ev,
		Data: data,
	}

	return nil
}

// Run try resent
func (l *launcher) reduce(keep int, du time.Duration) {
	conn := l.pool.Get()
	conn.Auth(client.Writable)
	// 保留最後 n 筆資料
	cache := list.New()
	for ca := range l.c {
		cache.PushBack(ca)
		for cache.Len() > keep {
			cache.Remove(cache.Front())
		}
		for {
			if err := conn.Fire(ca.Name, ca.Data); err == nil {
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
					pack := el.Value.(*client.Event)
					conn.Fire(pack.Name, pack.Data)
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
