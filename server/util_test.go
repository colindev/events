package main

import (
	"errors"
	"fmt"
	"testing"
)

func Test_makeLen(t *testing.T) {

	buf := makeLen('=', 12345)

	if buf.String() != "=12345\r\n" {
		t.Error("writeLen error:", buf.String())
	}
}

func Test_makeEvent(t *testing.T) {
	eventText := `aaa.bbb.ccc:{"prop1":123, "prop2": "xxx"}`
	expect := fmt.Sprintf(`=%d%s%s`, len(eventText), "\r\n", eventText)

	buf := makeEvent(eventText)
	if buf.String() != expect {
		t.Errorf("expect %s. but %s", expect, buf.String())
	}
}

func Test_makePong(t *testing.T) {
	pingText := `111 222 333 444
555 666`
	expect := fmt.Sprintf(`=%d%s%s`, len(pingText), "\r\n", pingText)

	buf := makeEvent(pingText)
	if buf.String() != expect {
		t.Errorf("expect %s. but %s", expect, buf.String())
	}
}

func Test_makeReply(t *testing.T) {
	replyText := `abc Ok`
	expect := fmt.Sprintf(`*%s`, replyText)

	buf := makeReply(replyText)
	if buf.String() != expect {
		t.Errorf("expect %s. but %s", expect, buf.String())
	}
}

func Test_makeError(t *testing.T) {
	err := errors.New("test error\r\n\t123\r\n\t456")
	expect := fmt.Sprintf(`!%d%s%s`, len(err.Error()), "\r\n", err.Error())

	buf := makeError(err)
	if buf.String() != expect {
		t.Errorf("expect %s. but %s", expect, buf.String())
	}
}
