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
	go l.reduce(time.Millisecond * 300)

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
func (l *launcher) reduce(du time.Duration) {
	conn := l.pool.Get()
	conn.Auth(client.Writable)
	for ca := range l.c {
		for {
			if err := conn.Fire(ca.Name, ca.Data); err == nil {
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
