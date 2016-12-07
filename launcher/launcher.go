package launcher

import (
	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
)

type (
	// Launcher responsible for event data
	Launcher interface {
		Fire(event.Event, event.RawData) error
	}

	launcher struct {
		pool client.Pool
	}
)

// New return a Launcher instance
func New(pool client.Pool) Launcher {
	return &launcher{pool}
}

func (l *launcher) Fire(ev event.Event, rd event.RawData) (err error) {

	conn := l.pool.Get()
	defer conn.Close()
	conn.Auth(client.Writable)

	return conn.Fire(ev, rd)
}
