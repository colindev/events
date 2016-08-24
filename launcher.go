package events

import (
	"encoding/json"

	"github.com/colindev/events/redis"
)

type (
	Launcher interface {
		Fire(Event, interface{}) error
	}

	launcher struct {
		pool redis.Pool
	}
)

func NewLauncher(pool redis.Pool) Launcher {
	return &launcher{pool}
}

func (l *launcher) Fire(ev Event, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	conn := l.pool.Get()
	defer conn.Close()

	_, err = conn.Do("PUBLISH", string(ev), string(b))

	return err
}
