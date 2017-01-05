package main

import "testing"

func TestFollowConn(t *testing.T) {

	var f interface{}
	f = &followConn{Conn: &conn{}}

	if _, ok := f.(Conn); !ok {
		t.Error("fail")
	}
}
