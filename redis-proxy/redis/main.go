package redis

import (
	x "github.com/garyburd/redigo/redis"
)

type Conn x.Conn

func Dial(network, addr string) (Conn, error) {
	return x.Dial(network, addr)
}

type Pool struct {
	*x.Pool
}

func NewPool(f func() (Conn, error), idle int) *Pool {

	dial := func() (x.Conn, error) {
		conn, err := f()
		return x.Conn(conn), err
	}
	pool := x.NewPool(dial, idle)
	return &Pool{pool}
}
