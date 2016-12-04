package client

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/colindev/events/event"
)

func TestConnReceive(t *testing.T) {

	buf := bytes.NewBuffer([]byte{})
	c := &conn{
		w: bufio.NewWriter(buf),
		r: bufio.NewReader(buf),
	}
	c.Fire(eventName, eventData)
	e, err := c.Receive()

	if err != nil {
		t.Error(err)
		t.Skip("skip check data detail")
	}

	if e, ok := e.(*Event); ok {
		if string(e.Name) != string(eventName) {
			t.Error("event name error:", string(e.Name))
		}

		if !bytes.Equal(e.Data, eventData.Bytes()) {
			t.Error("event data error:\n", string(e.Data))
		}
	} else {
		t.Error("receive type error")
	}

}

func TestParseLen(t *testing.T) {
	n, err := parseLen([]byte("=12345678987654321"))
	if err != nil {
		t.Error(err)
	}
	if n != 12345678987654321 {
		t.Error("parseLen error:", n)
	}
}

func BenchmarkParseEvent(b *testing.B) {

	// 直接利用原本機制產生測試用資料
	buf := bytes.NewBuffer([]byte{})
	c := &conn{
		w: bufio.NewWriter(buf),
		r: bufio.NewReader(buf),
	}
	c.Fire(eventName, eventData)

	line, _ := c.readLine()
	n, _ := parseLen(line[1:])
	p := make([]byte, n)
	io.ReadFull(c.r, p)

	for i := 0; i < b.N; i++ {
		parseEvent(p)
	}
}

var (
	eventName = event.Event("stream.accountants")
	eventData = event.RawData(`
	{
		"a":123,
		"numbers":[10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001]
	}
	`)
)