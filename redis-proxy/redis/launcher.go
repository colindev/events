package redis

import (
	"github.com/colindev/events/event"
	x "github.com/garyburd/redigo/redis"
)

type (
	// Launcher responsible for event data
	Launcher interface {
		Fire(event.Event, event.RawData) error
		Close() error
	}

	launcher struct {
		pool *x.Pool
	}
)

// NewLauncher return a Launcher instance
func NewLauncher(pool *Pool) Launcher {
	return &launcher{pool.Pool}
}

func (l *launcher) Fire(ev event.Event, rd event.RawData) (err error) {

	conn := l.pool.Get()
	defer conn.Close()

	_, err = conn.Do("PUBLISH", ev.String(), rd.String())

	return err
}

func (l *launcher) Close() error {
	return l.pool.Close()
}
