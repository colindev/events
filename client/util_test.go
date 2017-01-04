package client

import (
	"bytes"
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
