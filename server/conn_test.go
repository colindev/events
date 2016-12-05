package main

import (
	"regexp"
	"testing"
	"time"
)

func TestConn_EachChannels(t *testing.T) {
	c := conn{
		chs: map[string]*regexp.Regexp{
			"a": nil,
			"b": nil,
			"c": nil,
		},
	}

	chs1 := c.EachChannels(func(ch string, _ *regexp.Regexp) string {
		t.Log(ch)
		if ch != "b" {
			return ch
		}
		return ""
	})

	t.Log(chs1)
	if len(chs1) != 2 {
		t.Error("filter error")
	}

}

func TestConn_IsListening(t *testing.T) {

	c := newConn(nil, time.Now())

	ind, err := c.Subscribe([]byte("game.*\r\n"))
	if err != nil {
		t.Error(err)
	}
	if ind != "game.*" {
		t.Error("convert channel name error:", ind)
	}

	if !c.IsListening("game.start") {
		t.Error("match test fail:", c.EachChannels())
	}
}
