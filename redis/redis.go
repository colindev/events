package redis

import "github.com/garyburd/redigo/redis"

type (
	// Pool adaptor for redis.Pool
	Pool interface {
		Get() redis.Conn
	}
	// Conn adaptor for redis.Conn
	Conn redis.Conn
	// DialOption adaptor for redis.DialOption
	DialOption redis.DialOption
)

// Dial adaptor for redis.Dial
func Dial(network, address string, options ...DialOption) (Conn, error) {
	opts := []redis.DialOption{}
	for _, opt := range options {
		opts = append(opts, redis.DialOption(opt))
	}
	return redis.Dial(network, address)
}

// NewPool adaptor for redis.NewPool
func NewPool(fn func() (Conn, error), maxIdle int) Pool {
	return redis.NewPool(func() (redis.Conn, error) {
		conn, err := fn()
		return redis.Conn(conn), err
	}, maxIdle)
}
