package main

import (
	"testing"
	"time"
)

func TestConn_IsListening(t *testing.T) {

	c := newConn(nil, time.Now())

	ind, err := c.subscribe([]byte("game.*\r\n"))
	if err != nil {
		t.Error(err)
	}

	if !c.isListening("game.start") {
		t.Error("match test fail:", c.chs[ind])
	}
}
