package launcher

import (
	"testing"

	"github.com/colindev/events/redis"
)

func TestForge(t *testing.T) {
	t.Log(New(redis.NewPool(func() (redis.Conn, error) { return redis.Dial("tcp", "127.0.0.1:6379") }, 10)))
}
