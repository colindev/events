package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
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

func TestConn_ReadLine(t *testing.T) {

	var (
		r  io.Reader
		br *bufio.Reader
		c  Conn
	)

	temp := []byte{'A', 'B', 'C'}
	r = strings.NewReader(fmt.Sprintf("%s\r\n%s\n", temp, temp))
	br = bufio.NewReader(r)

	for i := 2; i > 0; i-- {
		c = &conn{r: br}
		line, err := c.ReadLine()
		if err != nil {
			t.Error(err)
		}
		t.Logf("readed %s", line)
		if !bytes.Equal(temp, line) {
			t.Errorf("read fail: [%b] != [%b]", temp, line)
		}
	}
}

func TestConnSubAndUnsub(t *testing.T) {
	c := newConn(nil, time.Now())

	temp := "aaa.*"
	tempSpace := fmt.Sprintf(" %s ", temp)

	ind, err := c.Subscribe([]byte(tempSpace))
	if err != nil {
		t.Error("subscribe error: ", err)
	}
	if ind != temp {
		t.Error("trim space fail: ", ind)
	}
	ind, err = c.Subscribe([]byte(temp))
	if err != nil {
		t.Error("duplicate sub error")
	}
	if ind != temp {
		t.Error("duplicate trim error:", ind)
	}

	if len(c.EachChannels()) != 1 {
		t.Error("sub fail")
		t.Skip("註冊錯誤後面不用測了")
	}

	// start test unsub
	otherChs := []string{"aa.*", "aaaa.*", "aaa.bbb"}
	for _, ch := range otherChs {
		c.Subscribe([]byte(ch))
	}

	ind, err = c.Unsubscribe([]byte(tempSpace))
	if err != nil {
		t.Error("unsubscribe error: ", err)
	}
	if ind != temp {
		t.Error("trim space fail: ", ind)
	}
	ind, err = c.Unsubscribe([]byte(temp))
	if err == nil {
		t.Error("duplicate unsub error")
	}
	if ind != "" {
		t.Error("duplicate unsub drop ret fail: ", ind)
	}

	curChs := c.EachChannels()
	for _, ch := range otherChs {
		if _, exists := curChs[ch]; !exists {
			t.Error("miss channel: ", ch)
		}
	}

}

func TestConn_writeLen(t *testing.T) {
	w := bytes.NewBuffer(nil)
	bw := bufio.NewWriter(w)
	c := &conn{w: bw}

	c.writeLen('=', 12345)
	if err := bw.Flush(); err != nil {
		t.Error("buf flush error: ", err)
	}

	if w.String() != "=12345\r\n" {
		t.Error("writeLen error:", w.String())
	}
}

func checkBuffer(name string, t *testing.T, w *bytes.Buffer, bw *bufio.Writer, expect string) {

	if err := bw.Flush(); err != nil {
		t.Errorf("%s buf flush error: %v", name, err)
	}
	if w.String() != expect {
		t.Errorf("%s error: expect[%s], but[%s]", name, expect, w.String())
	}
}

func TestConn_writeEvent(t *testing.T) {
	eventText := `aaa.bbb.ccc:{"prop1":123, "prop2": "xxx"}`
	expect := fmt.Sprintf(`=%d%s%s`, len(eventText), "\r\n", eventText)

	w := bytes.NewBuffer(nil)
	bw := bufio.NewWriter(w)
	c := &conn{w: bw}

	c.writeEvent(eventText)
	checkBuffer("writeEvent", t, w, bw, expect)
}

func TestConn_writePong(t *testing.T) {
	pingText := `111 222 333 444
555 666`
	expect := fmt.Sprintf(`=%d%s%s`, len(pingText), "\r\n", pingText)

	w := bytes.NewBuffer(nil)
	bw := bufio.NewWriter(w)
	c := &conn{w: bw}

	c.writeEvent(pingText)
	checkBuffer("writePong", t, w, bw, expect)
}

func TestConn_writeReply(t *testing.T) {
	replyText := `abc Ok`
	expect := fmt.Sprintf(`*%s`, replyText)

	w := bytes.NewBuffer(nil)
	bw := bufio.NewWriter(w)
	c := &conn{w: bw}

	c.writeReply(replyText)
	checkBuffer("writeReply", t, w, bw, expect)

}
