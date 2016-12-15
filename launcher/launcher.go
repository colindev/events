package launcher

import (
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
		c    chan *cache
	}

	cache struct {
		event.Event
		event.RawData
	}
)

// New return a Launcher instance
func New(pool client.Pool) Launcher {
	l := &launcher{
		pool: pool,
		c:    make(chan *cache, 100),
	}

	l.wg.Add(1)
	go l.reduce(time.Millisecond)

	return l
}

func (l *launcher) Fire(ev event.Event, rd event.RawData) (err error) {

	l.c <- &cache{
		Event:   ev,
		RawData: rd,
	}

	return nil
}

// Run try resent
func (l *launcher) reduce(du time.Duration) {
	conn := l.pool.Get()
	conn.Auth(client.Writable)
	for ca := range l.c {
		for {
			if err := conn.Fire(ca.Event, ca.RawData); err == nil {
				break
			}
			conn.Close()
			time.Sleep(du)
			conn = l.pool.Get()
			conn.Auth(client.Writable)
		}
	}

	l.wg.Done()
}

func (l *launcher) Close() error {
	close(l.c)
	l.wg.Wait()
	return nil
}
