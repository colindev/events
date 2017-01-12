package connection

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/colindev/events/event"
)

func Test_ParseLen(t *testing.T) {

	i, err := ParseLen([]byte("12345600"))
	if err != nil {
		t.Error("parseLen error: ", err)
	}

	if i != 12345600 {
		t.Error("parseLen fail:", i)
	}

}

func Test_ParseTargetAndLen(t *testing.T) {
	if n, i, e := ParseTargetAndLen([]byte("aaa:123")); n != "aaa" || i != 123 || e != nil {
		t.Error("ParseTargetAndLen(aaa:123) error", n, i, e)
	}
	if n, i, e := ParseTargetAndLen([]byte(":123")); n != "" || i != 123 || e != nil {
		t.Error("ParseTargetAndLen(:123) error", n, i, e)
	}
}

func Test_ParseEvent(t *testing.T) {

	eventName := []byte("aaa.bbb.ccc")
	eventData := []byte("123456789\n123456789\r\n123456789")

	buf := bytes.NewBuffer(nil)
	buf.Write(eventName)
	buf.WriteByte(':')
	buf.Write(eventData)

	name, data, err := ParseEvent(buf.Bytes())
	if err != nil {
		t.Error("parseEvent error: ", err)
		t.Skip("skip check parse ret")
	}

	if !bytes.Equal(name.Bytes(), eventName) {
		t.Errorf("parseEvent name error expect [%s], but [%s]", eventName, name)
	}
	if !bytes.Equal(data, eventData) {
		t.Errorf("parseEvent data error expect [%s], but [%s]", eventData, data)
	}

}

func Test_MakeEventStream(t *testing.T) {

	eventName := "aaa.bbb.ccc"
	eventData := "123456789\r\n123456789\r\n123456789\r"
	expect := []byte(eventName + ":" + eventData)

	p := MakeEventStream(event.Event(eventName), event.RawData(eventData))

	if !bytes.Equal(p, expect) {
		t.Errorf("makeEventStream error expect [%s], but [%s]", expect, p)
	}
}

func Test_ParseSinceUntil(t *testing.T) {
	if s, u := ParseSinceUntil([]byte("123:")); s != 123 || u != 0 {
		t.Error("ParseSinceUntil(123:) fail", s, u)
	}
	if s, u := ParseSinceUntil([]byte("123:456")); s != 123 || u != 456 {
		t.Error("ParseSinceUntil(123:456) fail", s, u)
	}
	if s, u := ParseSinceUntil([]byte(":456")); s != 0 || u != 456 {
		t.Error("ParseSinceUntil(:456) fail", s, u)
	}
}

func Test_ReadLine(t *testing.T) {
	var (
		err  error
		line []byte
	)

	r := bytes.NewBuffer(nil)
	buf := bufio.NewReader(r)

	_, err = ReadLine(buf)
	if err != io.EOF {
		t.Error("EOL")
	}

	r.Write([]byte{'\r', '\n'})
	line, err = ReadLine(buf)
	if !bytes.Equal(line, []byte{}) {
		t.Errorf("expect empty line, but %+v", line)
	}

	r.Write([]byte("123\r\n456\r\n"))

	ReadLine(buf)
	line, _ = ReadLine(buf)
	if string(line) != "456" {
		t.Errorf("expect 456, but [%s]", line)
	}
}

func Test_ReadLen(t *testing.T) {

	temp := "1234567890"
	tempStream := fmt.Sprintf("%sx\r\n", temp)

	buf := bytes.NewBuffer(nil)
	r := bufio.NewReader(buf)
	buf.Write([]byte(tempStream))

	p, err := ReadLen(r, []byte{'1', '0'})
	if err != nil {
		t.Error("readLen error: ", err)
		t.Skip("skip check contents")
	}

	if string(p) != temp {
		t.Errorf("readLen error, expect [%s], but [%s]", temp, p)
	}

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

func Test_WriteLen(t *testing.T) {

	prefix := byte('@')
	length := 12345
	expect := fmt.Sprintf("%c%d\r\n", prefix, length)

	buf := bytes.NewBuffer(nil)
	w := bufio.NewWriter(buf)
	WriteLen(w, prefix, length)

	checkBuf("writeLen", t, buf, w, expect)
}

func Test_WriteEvent(t *testing.T) {

	prefix := CEvent
	eventText := "aaa.bbb:ccc"
	expect := fmt.Sprintf("%c%d\r\n%s", prefix, len(eventText), eventText)

	buf := bytes.NewBuffer(nil)
	w := bufio.NewWriter(buf)
	WriteEvent(w, []byte(eventText))

	checkBuf("writeEvent", t, buf, w, expect)
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

func BenchmarkParseEvent(b *testing.B) {

	// 直接利用原本機制產生測試用資料
	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()
	r := bufio.NewReader(pr)
	w := bufio.NewWriter(pw)

	N := b.N
	go func() {
		for i := 0; i < N; i++ {
			WriteEvent(w, MakeEventStream(eventName, eventData))
			w.Flush()
		}
	}()

	for i := 0; i < N; i++ {
		line, err := ReadLine(r)

		if len(line) == 0 {
			b.Skip(i, string(line), err)
		}

		n, _ := ParseLen(line[1:])
		p := make([]byte, n)
		io.ReadFull(r, p)
		ParseEvent(p)
	}
}
