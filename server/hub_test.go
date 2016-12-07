package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/colindev/events/server/fake"
)

func createHub(t *testing.T) *Hub {

	hub, err := NewHub(&Env{
		Debug:      true,
		AuthDSN:    ":file::memory:",
		EventDSN:   ":file::memory:",
		GCDuration: "1h",
	}, log.New(os.Stdout, "", log.LstdFlags))
	if err != nil {
		t.Error("NewHub error:", err)
		t.Skip()
	}

	return hub
}

func TestHub_auth(t *testing.T) {
	hub := createHub(t)

	c := &conn{}
	if err := hub.auth(c, []byte("test")); err == nil {
		t.Error("auth stream has wrong Read Writer Flags, but passed")
		t.Skip()
	}
	if err := hub.auth(c, []byte(":test")); err == nil {
		t.Error("auth stream has wrong Read Writer Flags, but passed")
		t.Skip()
	}
	if err := hub.auth(c, []byte("test:3")); err != nil {
		t.Error("hub.auth error: ", err)
	}
	if err := hub.auth(&conn{}, []byte("test:3")); err == nil {
		t.Error("can't duplicate auth")
	}
	if c.GetName() != "test" {
		t.Errorf("auth set conn name error: expect %s, but %s", "test", c.GetName())
	}

	if cc, exists := hub.m["test"]; !exists {
		t.Errorf("h.m not found [%s]", "test")
	} else if cc != c {
		t.Errorf("found wrong conn pointer: %+v", cc)
	}

	if err := hub.auth(c, []byte(":3")); err != nil {
		t.Error("ghost auth error:", err)
	}
	if !hub.g[c] {
		t.Error("miss hub.g[c]")
	}

	t.Log("[ghost]", hub.g)
	t.Log(hub.m)

}

func TestHub_quit(t *testing.T) {
	hub := createHub(t)
	now := time.Now()

	c := &conn{conn: &fake.NetConn{}}
	hub.auth(c, []byte("test"))

	auth := hub.quit(c, now)

	if auth.DisconnectedAt != now.Unix() {
		t.Errorf("hub.quit must set disconnected_at to conn %+v", auth)
	}
}

func TestHub_quitAll(t *testing.T) {
	hub := createHub(t)

	c := &conn{conn: &fake.NetConn{}}
	hub.auth(&conn{conn: &fake.NetConn{}}, []byte("test1:1"))
	hub.auth(c, []byte("test2:1"))
	hub.auth(c, []byte("test3:3"))

	t.Log("auth:", hub.m)
	t.Log("ghost:", hub.g)

	hub.quitAll(time.Now())

	if len(hub.m) != 0 {
		t.Error("GC auth conn fail", hub.m)
	}
	if len(hub.g) != 0 {
		t.Error("GC ghost conn fail", hub.g)
	}
}
