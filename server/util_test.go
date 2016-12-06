package main

import (
	"bytes"
	"testing"
)

func Test_parseLen(t *testing.T) {

	i, err := parseLen([]byte("12345600"))
	if err != nil {
		t.Error("parseLen error: ", err)
	}

	if i != 12345600 {
		t.Error("parseLen fail:", i)
	}

}

func Test_parseEvent(t *testing.T) {

	eventName := []byte("aaa.bbb.ccc")
	eventData := []byte("123456789\n123456789\r\n123456789")

	buf := bytes.NewBuffer(nil)
	buf.Write(eventName)
	buf.WriteByte(':')
	buf.Write(eventData)

	name, data, err := parseEvent(buf.Bytes())
	if err != nil {
		t.Error("parseEvent error: ", err)
		t.Skip("skip check parse ret")
	}

	if !bytes.Equal(name, eventName) {
		t.Errorf("parseEvent name error expect [%s], but [%s]", eventName, name)
	}
	if !bytes.Equal(data, eventData) {
		t.Errorf("parseEvent data error expect [%s], but [%s]", eventData, data)
	}

}

func Test_makeEventStream(t *testing.T) {

	eventName := "aaa.bbb.ccc"
	eventData := "123456789\r\n123456789\r\n123456789\r"
	expect := []byte(eventName + ":" + eventData)

	p := makeEventStream([]byte(eventName), []byte(eventData))

	if !bytes.Equal(p, expect) {
		t.Errorf("makeEventStream error expect [%s], but [%s]", expect, p)
	}

}
