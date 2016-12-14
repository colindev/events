package main

import "testing"

func Test_parseNotifyer(t *testing.T) {

	var (
		n   *Notifyer
		err error
	)

	_, err = parseNotifyer("xxx")
	if err == nil {
		t.Error("parseNotifyer fail")
	}

	n, err = parseNotifyer("redis://xxx")
	if err != nil {
		t.Error(err)
	}
	if x, ok := n.from.(*listener); !ok {
		t.Errorf("%#v", x)
	}
	if n.fromType != "redis" {
		t.Error("fromType error:", n.fromType)
	}

	n, err = parseNotifyer("events://xxxxx")
	if err != nil {
		t.Error(err)
	}
	if n.fromType != "events" {
		t.Error("fromType error:", n.fromType)
	}
	t.Logf("%#v", n)

}

func Test_parseReceiver(t *testing.T) {

	var (
		mode string
		err  error
	)

	_, _, err = parseReceiver("xxxxxx")
	if err == nil {
		t.Error("parseReceiver error")
	}

	_, mode, _ = parseReceiver("redis://xxxxx")
	if mode != "redis" {
		t.Error("parse redis receiver error:", mode)
	}

	_, mode, _ = parseReceiver("events://xxxxxx")
	if mode != "events" {
		t.Error("parse events receiver error:", mode)
	}
}
