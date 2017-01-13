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

	"github.com/colindev/events/connection"
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
	c.SetFlags(connection.Writable)
	if !c.Writable() {
		t.Error("this conn is writable")
	}
	c.w.WriteString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err := c.w.Flush(); err != nil {
		t.Error(err)
	}

	// read only client/conn w 為正確的net.Conn / Conn.Writable 應該回傳 false 表示 Conn.Recieve 的 Event stream 會被丟棄
	c.SetFlags(connection.Readable)
	c.conn.(*fake.NetConn).W = func([]byte) (int, error) { return 0, io.EOF }
	if c.Writable() {
		t.Error("this conn can't write")
	}
	c.w.WriteString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err := c.w.Flush(); err != io.EOF {
		t.Error(err)
	}

	c.SetFlags(connection.Writable | connection.Readable)
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

	ch := c.Subscribe("game.*")
	if ch != "game.*" {
		t.Error("convert channel name error:", ch)
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
	ch := c.Subscribe(temp)
	if ch != temp {
		t.Error("sub return error: ", ch)
	}

	if len(c.EachChannels()) != 1 {
		t.Error("sub fail")
		t.Skip("註冊錯誤後面不用測了")
	}

	// start test unsub
	otherChs := []event.Event{"aa.*", "aaaa.*", "aaa.bbb"}
	for _, ch := range otherChs {
		c.Subscribe(ch.String())
	}

	ch = c.Unsubscribe(temp)
	if ch != temp {
		t.Error("unsub return fail: ", ch)
	}
	ch = c.Unsubscribe(temp)
	if ch != "" {
		t.Error("duplicate unsub return fail")
	}

	curChs := c.EachChannels()
	for _, ch := range otherChs {
		if _, exists := curChs[ch]; !exists {
			t.Error("miss channel: ", ch)
		}
	}

}

func TestConn_Receive(t *testing.T) {

	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()

	c := &conn{
		r: bufio.NewReader(pr),
		w: bufio.NewWriter(pw),
	}
	var (
		m   Message
		chs = []string{"a", "b", "c"}
	)

	go func() {

		connection.WriteAuth(c.w, "test", 3)
		c.flush(connection.EOL)

		connection.WriteRecover(c.w, 123, 456)
		c.flush(connection.EOL)

		connection.WriteSubscribe(c.w, chs...)
		c.flush(nil)

		connection.WriteUnsubscribe(c.w, chs...)
		c.flush(nil)

		connection.WritePing(c.w, "aaaa")
		c.flush(connection.EOL)

		connection.WriteEvent(c.w, connection.MakeEventStream("test.1", event.RawData("xxxxx")))
		c.flush(connection.EOL)

		connection.WriteEventTo(c.w, "hello", connection.MakeEventStream("test.2", event.RawData("xxxxx")))
		c.flush(connection.EOL)

	}()

	// auth
	m = c.Receive()
	if m.Error != nil {
		t.Error(m.Error)
	} else if v, ok := m.Value.(MessageAuth); !ok {
		t.Errorf("auth read fail %#v", m.Value)
	} else if v.Name != "test" {
		t.Errorf("auth name fail %#v", v.Name)
	} else if v.Flags != 3 {
		t.Errorf("auth flags fail %#v", v.Flags)
	}

	// recover
	m = c.Receive()
	if m.Error != nil {
		t.Error(m.Error)
	} else if v, ok := m.Value.(MessageRecover); !ok {
		t.Errorf("recover read fail %#v", m.Value)
	} else if v.Since != 123 || v.Until != 456 {
		t.Errorf("recover %d %d", v.Since, v.Until)
	}

	// subscribe
	for _, ch := range chs {
		m = c.Receive()
		if m.Error != nil {
			t.Error(m.Error)
		} else if v, ok := m.Value.(MessageSubscribe); !ok {
			t.Errorf("sub read fail %#v", m.Value)
		} else if v.Channel != ch {
			t.Errorf("sub error expect %s, but %s", ch, v.Channel)
		}
	}

	// unsubscribe
	for _, ch := range chs {
		m = c.Receive()
		if m.Error != nil {
			t.Error(m.Error)
		} else if v, ok := m.Value.(MessageUnsubscribe); !ok {
			t.Errorf("unsub read fail %#v", m.Value)
		} else if v.Channel != ch {
			t.Errorf("unsub error expect %s, but %s", ch, v.Channel)
		}
	}

	// ping
	m = c.Receive()
	if m.Error != nil {
		t.Error(m.Error)
	} else if v, ok := m.Value.(MessagePing); !ok {
		t.Errorf("ping read fail %#v", m.Value)
	} else if string(v.Payload) != "aaaa" {
		t.Errorf("ping error %s", v.Payload)
	}

	// event
	m = c.Receive()
	if m.Error != nil {
		t.Error(m.Error)
	} else if v, ok := m.Value.(MessageEvent); !ok {
		t.Errorf("event read fail %#v", m.Value)
	} else if v.To != "" {
		t.Errorf("event error %#v", v)
	} else if v.Name != "test.1" {
		t.Errorf("event error %#v", v)
	} else if v.RawData.String() != "xxxxx" {
		t.Errorf("event error %#v", v)
	}

	// event to
	m = c.Receive()
	if m.Error != nil {
		t.Error(m.Error)
	} else if v, ok := m.Value.(MessageEvent); !ok {
		t.Errorf("event to read fail %#v", m.Value)
	} else if v.To != "hello" {
		t.Errorf("event to error %#v", v)
	} else if v.Name != "test.2" {
		t.Errorf("event to error %#v", v)
	} else if v.RawData.String() != "xxxxx" {
		t.Errorf("event to error %#v", v)
	}

}
