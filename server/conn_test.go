package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/server/fake"
)

func TestNewConnAndGetAuth(t *testing.T) {
	now := time.Now()
	c := newConn(nil, now)

	if auth := c.GetAuth(); auth.ConnectedAt != now.Unix() {
		t.Error("conn auth connected_at error", auth)
	}
}

func TestConnName(t *testing.T) {
	c := &conn{}

	name := "abc"
	c.SetName(name)

	if !c.HasName() {
		t.Error("Conn.HasName return false")
	}
	if s := c.GetName(); s != name {
		t.Errorf("Conn.GetName error: expect %s, but %s", name, s)
	}
}

func TestConn_SetFlags(t *testing.T) {
	c := &conn{conn: &fake.NetConn{}}

	// no flags
	c.SetFlags(0)
	c.conn.(*fake.NetConn).W = func([]byte) (int, error) { return 0, errors.New("替換 write buffer 到 ioutil discard 失敗") }
	if c.Writable() {
		t.Error("this conn can't write")
	}
	c.w.WriteString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err := c.w.Flush(); err != nil {
		t.Error(err)
	}

	// write only client/conn 應該被替換成 ioutil.Discard / Conn.Writable 應該是true 表示接收 Conn.Receive 傳來的 stream
	c.SetFlags(client.Writable)
	if !c.Writable() {
		t.Error("this conn is writable")
	}
	c.w.WriteString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err := c.w.Flush(); err != nil {
		t.Error(err)
	}

	// read only client/conn w 為正確的net.Conn / Conn.Writable 應該回傳 false 表示 Conn.Recieve 的 Event stream 會被丟棄
	c.SetFlags(client.Readable)
	c.conn.(*fake.NetConn).W = func([]byte) (int, error) { return 0, io.EOF }
	if c.Writable() {
		t.Error("this conn can't write")
	}
	c.w.WriteString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err := c.w.Flush(); err != io.EOF {
		t.Error(err)
	}

	c.SetFlags(client.Writable | client.Readable)
	if !c.Writable() {
		t.Error("SetFlag(W|R) but Conn.Writable() return false")
	}

}

func TestConn_EachChannels(t *testing.T) {
	c := &conn{
		chs: map[event.Event]bool{
			"a": true,
			"b": true,
			"c": true,
		},
	}

	chs1 := c.EachChannels(func(ch event.Event) event.Event {
		t.Log(ch)
		if ch.String() != "b" {
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
	otherChs := []event.Event{"aa.*", "aaaa.*", "aaa.bbb"}
	for _, ch := range otherChs {
		c.Subscribe(ch.Bytes())
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

func createWBWC() (*bytes.Buffer, *bufio.Writer, *conn) {
	w := bytes.NewBuffer(nil)
	bw := bufio.NewWriter(w)
	return w, bw, &conn{w: bw}
}
func checkBuffer(name string, t *testing.T, w *bytes.Buffer, bw *bufio.Writer, expect string) {

	if err := bw.Flush(); err != nil {
		t.Errorf("%s buf flush error: %v", name, err)
	}
	if w.String() != expect {
		t.Errorf("%s error: expect[%s], but[%s]", name, expect, w.String())
	}
}
