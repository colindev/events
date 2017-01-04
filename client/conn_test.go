package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/colindev/events/event"
)

func createBufReader(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}

func createConn(s string) *conn {
	return &conn{r: createBufReader(s)}
}

func TestFlag_String(t *testing.T) {

	for s, i := range map[string]int{"rw": 3, "-w": 1, "r-": 2, "--": 0} {
		if ss := Flag(i).String(); ss != s {
			t.Errorf("Flag String error %d expect %s, but %s", i, s, ss)
		}
	}
}

func TestConn_readLine(t *testing.T) {
	var (
		c    *conn
		err  error
		line []byte
	)

	c = createConn("")
	_, err = c.readLine()
	if err != io.EOF {
		t.Error("EOL")
	}

	c.r = createBufReader("\n")
	_, err = c.readLine()
	if err == nil {
		t.Error("miss \\r")
	}

	c.r = createBufReader("\r\n")
	line, err = c.readLine()
	if !bytes.Equal(line, []byte{}) {
		t.Errorf("expect empty line, but %+v", line)
	}

	c.r = createBufReader("123\r\n456\r\n")

	c.readLine()
	line, _ = c.readLine()
	if string(line) != "456" {
		t.Errorf("expect 456, but [%s]", line)
	}
}

func TestConn_readLen(t *testing.T) {

	c := &conn{}

	temp := "1234567890"
	tempStream := fmt.Sprintf("%sx\r\n", temp)

	c.r = createBufReader(tempStream)

	p, err := c.readLen([]byte{'1', '0'})
	if err != nil {
		t.Error("readLen error: ", err)
		t.Skip("skip check contents")
	}

	if string(p) != temp {
		t.Errorf("readLen error, expect [%s], but [%s]", temp, p)
	}

}

func TestConn_ReceiveReply(t *testing.T) {

	replyText := "ok 123!*$"
	replyStream := fmt.Sprintf(`* %s %s`, replyText, "\r\n")

	c := createConn(replyStream)

	ret, err := c.Receive()

	if err != nil {
		t.Error(err)
		t.Skip("skip check data detail")
	}

	if ret, ok := ret.(*Reply); ok {
		if s := ret.String(); s != replyText {
			t.Errorf("reply expect [%s], but [%s]", replyText, s)
		}
	} else {
		t.Error("receive type error")
	}
}

func TestConn_ReceiveErr(t *testing.T) {

	errText := "some thing error\r\n\t123\r\n\t456"
	errStream := fmt.Sprintf(`!%d%s%s%s`, len(errText), "\r\n", errText, "\r\n")

	c := createConn(errStream)

	ret, err := c.Receive()

	if ret != nil {
		t.Error("conn receive error can't with ret:", ret)
	}

	if err == nil {
		t.Error("conn receive error, but return nil")
		t.Skip("skip check data detail")
	}

	if err.Error() != errText {
		t.Errorf("receive error expect [%s], but [%s]", errText, err.Error())
	}
}

func TestConn_ReceivePong(t *testing.T) {

	pongText := "123\r\n456\r\n789"
	pongStream := fmt.Sprintf(`@%d%s%s%s`, len(pongText), "\r\n", pongText, "\r\n")

	c := createConn(pongStream)

	ret, err := c.Receive()
	if err != nil {
		t.Error(err)
		t.Skip("skip check data detail")
	}

	if ret, ok := ret.(*Event); ok {
		if ret.Name != event.PONG {
			t.Errorf("expect event name [%s], but [%s]", event.PONG, ret.Name)
		}

		if ret.Data.String() != pongText {
			t.Errorf("expect event data [%s], but [%s]", pongText, ret.Data)
		}
	} else {
		t.Error("receive type error")
	}

}

var (
	// use to test ReceiveEvent, benchmark
	eventName = event.Event("stream.accountants")
	eventData = event.RawData(`
	{
		"a":123,
		"numbers":[10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001,10000001]
	}
	`)
)

func TestConn_ReceiveEvent(t *testing.T) {

	b, err := event.Compress(eventData)
	if err != nil {
		t.Error(err)
		t.Skip("compress fail")
	}
	eventText := fmt.Sprintf(`%s:%s`, eventName, b)
	eventStream := fmt.Sprintf(`=%d%s%s%s`, len(eventText), "\r\n", eventText, "\r\n")

	c := createConn(eventStream)

	ret, err := c.Receive()

	if err != nil {
		t.Error(err)
		t.Skip("skip check data detail")
	}

	if ev, ok := ret.(*Event); ok {
		if string(ev.Name) != string(eventName) {
			t.Error("event name error:", string(ev.Name))
		}

		if !bytes.Equal(ev.Data, eventData.Bytes()) {
			t.Error("event data error:\n", string(ev.Data))
		}
	} else {
		t.Errorf("receive type error: expect *Event but [%#v]", ret)
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
	n, _ := ParseLen(line[1:])
	p := make([]byte, n)
	io.ReadFull(c.r, p)

	for i := 0; i < b.N; i++ {
		ParseEvent(p)
	}
}

func createBWC() (*bytes.Buffer, *bufio.Writer, *conn) {
	buf := bytes.NewBuffer(nil)
	bw := bufio.NewWriter(buf)
	return buf, bw, &conn{w: bw}
}

func checkBuf(name string, t *testing.T, buf *bytes.Buffer, bw *bufio.Writer, expect string) {
	if err := bw.Flush(); err != nil {
		t.Error("flush error:", err)
		t.Skip("skip check buf data")
	}

	if s := buf.String(); s != expect {
		t.Errorf("%s expect [%s], but [%s]", name, expect, s)
	}
}

func TestConn_writeLen(t *testing.T) {

	buf, bw, c := createBWC()

	prefix := byte('@')
	length := 12345
	expect := fmt.Sprintf("%c%d\r\n", prefix, length)

	c.writeLen(prefix, length)

	checkBuf("writeLen", t, buf, bw, expect)
}

func TestConn_writeEvent(t *testing.T) {
	buf, bw, c := createBWC()

	prefix := CEvent
	eventText := "aaa.bbb:ccc"
	expect := fmt.Sprintf("%c%d\r\n%s", prefix, len(eventText), eventText)

	c.writeEvent([]byte(eventText))

	checkBuf("writeEvent", t, buf, bw, expect)
}

func TestConn_Auth(t *testing.T) {
	buf, bw, c := createBWC()

	prefix := CAuth
	authText := "test name"
	flags := Writable | Readable
	expect := fmt.Sprintf("%c%s:%d\r\n", prefix, authText, 3)

	c.name = authText
	c.Auth(flags)

	checkBuf("Auth", t, buf, bw, expect)
}

func TestConn_Recover(t *testing.T) {
	buf, bw, c := createBWC()

	prefix := CRecover
	since := int64(12345600)
	until := int64(0)
	expect := fmt.Sprintf("%c%d:%d\r\n", prefix, since, until)

	c.Recover(since, until)

	checkBuf("RecoverSince", t, buf, bw, expect)
}

func TestConn_Subscribe(t *testing.T) {
	buf, bw, c := createBWC()

	prefix := CAddChan
	chanName := "xyz"
	expect := fmt.Sprintf("%c%s\r\n", prefix, chanName)

	c.Subscribe(chanName)

	checkBuf("Subscribe", t, buf, bw, expect)
}

func TestConn_Unsubscribe(t *testing.T) {
	buf, bw, c := createBWC()

	prefix := CDelChan
	chanName := "xyz"
	expect := fmt.Sprintf("%c%s\r\n", prefix, chanName)

	c.Unsubscribe(chanName)

	checkBuf("Unsubscribe", t, buf, bw, expect)
}

func TestConn_Fire(t *testing.T) {
	buf, bw, c := createBWC()

	prefix := CEvent
	b, err := event.Compress(eventData)
	if err != nil {
		t.Error(err)
		t.Skip("compress fail")
	}
	eventText := fmt.Sprintf("%s:%s", eventName, b)
	expect := fmt.Sprintf("%c%d\r\n%s\r\n", prefix, len(eventText), eventText)

	c.Fire(eventName, eventData)

	checkBuf("Fire", t, buf, bw, expect)
}

func TestConn_Ping(t *testing.T) {
	buf, bw, c := createBWC()

	prefix := CPing
	pingText := "123\n456\r\n789\r\n"
	expect := fmt.Sprintf("%c%d\r\n%s\r\n", prefix, len(pingText), pingText)

	c.Ping(pingText)

	checkBuf("Ping", t, buf, bw, expect)
}
