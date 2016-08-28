package redis

import (
	"errors"

	"github.com/garyburd/redigo/redis"
)

type (
	// Pool adaptor for redis.Pool
	Pool interface {
		Get() redis.Conn
	}
	// Conn adaptor for redis.Conn
	Conn redis.Conn
	// DialOption adaptor for redis.DialOption
	DialOption redis.DialOption
	// PubSubConn is conn for redis PUB/SUB
	PubSubConn struct {
		redis.PubSubConn
	}
	// Message adapt redis.Message
	Message redis.Message
	// PMessage adapt redis.PMessage
	PMessage redis.PMessage
	// Subscription adapt redis.Subscription
	Subscription redis.Subscription
	// Pong adapt redis.Pong
	Pong redis.Pong
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

// ErrUndefined will be returned when redis.PubSubConn receive undefined message type
var ErrUndefined = errors.New("[events/redis] pub/sub undefined")

// Receive convert redis.PubSubConn.Receive return value type
func (c PubSubConn) Receive() interface{} {
	switch v := c.PubSubConn.Receive().(type) {
	case redis.Message:
		return Message(v)
	case redis.PMessage:
		return PMessage(v)
	case redis.Subscription:
		return Subscription(v)
	case redis.Pong:
		return Pong(v)
	case error:
		return v
	}

	return ErrUndefined
}
