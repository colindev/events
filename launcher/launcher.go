package launcher

import (
	"github.com/colindev/events/event"
	"github.com/colindev/events/redis"
)

type (
	// Launcher responsible for event data
	Launcher interface {
		Fire(event.Event, event.RawData) error
	}

	launcher struct {
		pool redis.Pool
	}
)

// New return a Launcher instance
func New(pool redis.Pool) Launcher {
	return &launcher{pool}
}

func (l *launcher) Fire(ev event.Event, rd event.RawData) (err error) {

	conn := l.pool.Get()
	defer conn.Close()

	_, err = conn.Do("PUBLISH", ev.String(), rd.String())

	return err
}
