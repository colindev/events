package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/colindev/events/connection"
	"github.com/colindev/events/event"
	"github.com/colindev/events/server/fake"
)

func createHub(t *testing.T) *Hub {

	hub, err := NewHub(&Env{
		Debug:      true,
		AuthDSN:    "file::memory:?cache=shared",
		EventDSN:   "file::memory:?cache=shared",
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
	if err := hub.auth(c, MessageAuth{Name: "test"}); err != nil {
		t.Error(err)
		t.Skip(hub.info(false))
	}
	if err := hub.auth(&conn{}, MessageAuth{Name: "test", Flags: 3}); err == nil {
		t.Error("can't duplicate auth")
		t.Skip(hub.info(false))
	}
	if c.GetName() != "test" {
		t.Errorf("auth set conn name error: expect %s, but %s", "test", c.GetName())
	}

	if cc, exists := hub.m["test"]; !exists {
		t.Errorf("h.m not found [%s]", "test")
	} else if cc != c {
		t.Errorf("found wrong conn pointer: %+v", cc)
	}

	if err := hub.auth(c, MessageAuth{Flags: 3}); err != nil {
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
	hub.auth(c, MessageAuth{Name: "test"})

	auth := hub.quit(c, now)

	if auth.DisconnectedAt != now.Unix() {
		t.Errorf("hub.quit must set disconnected_at to conn %+v", auth)
	}
}

func TestHub_quitAll(t *testing.T) {
	hub := createHub(t)

	c := &conn{conn: &fake.NetConn{}}
	hub.auth(&conn{conn: &fake.NetConn{}}, MessageAuth{Name: "test1", Flags: 1})
	hub.auth(c, MessageAuth{Name: "test2", Flags: 2})
	hub.auth(c, MessageAuth{Name: "test3", Flags: 3})

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

func BenchmarkHub_handle(b *testing.B) {

	N := 10

	hub, err := NewHub(&Env{
		Debug:      true,
		AuthDSN:    "file::memory:?cache=shared",
		EventDSN:   "file::memory:?cache=shared",
		GCDuration: "30h",
	}, log.New(ioutil.Discard, "bench", log.Lshortfile))
	if err != nil {
		b.Error(err)
		b.Skip()
	}

	sc := []*conn{}

	for i := 0; i < N; i++ {
		r, w := io.Pipe()
		defer w.Close()
		defer r.Close()
		cc := bufio.NewWriter(w)

		fNetConn := &fake.NetConn{
			W: func(p []byte) (int, error) {
				return len(p), nil
			},
			R: r,
		}

		c := &conn{
			conn: fNetConn,
			w:    bufio.NewWriter(fNetConn),
			r:    bufio.NewReader(fNetConn),
			chs:  map[event.Event]bool{},
		}

		sc = append(sc, c)

		go hub.handle(c)
		go func(i int) {
			connection.WriteAuth(cc, "", 3)
			cc.WriteByte('\n')
			connection.WriteSubscribe(cc, "*")
			cc.WriteByte('\n')
			cc.Flush()
		}(i)
	}

	rd, err := event.Compress(event.RawData(`{"game":{"series_id":308,"domain":"6","currency":"CNY","award":520,"state":3,"price":10,"expect_person":52,"person_limit":52,"auto_payment":true},"status":3,"wallet":{"cash_entry":[{"id":80447376,"cash_id":71825765,"user_id":148511787,"currency":"CNY","opcode":"10001","created_at":"2017-01-10T13:57:15+0800","amount":"520","memo":"308-6-CNY-520.000000-148511787-1484027821-22089394","ref_id":"","balance":944997.61,"operator":[],"tag":"","cash_version":202}]},"order_payload":{"user_id":148511787,"username":"toyota999","entry_id":80447373,"parents_id":[6],"numbers":[10000001,10000010,10000020,10000012,10000019,10000045,10000032,10000022,10000018,10000033,10000051,10000028,10000040,10000024,10000026,10000039,10000005,10000006,10000002,10000041,10000003,10000009,10000043,10000029,10000030,10000048,10000015,10000008,10000050,10000037,10000034,10000046,10000044,10000049,10000014,10000052,10000011,10000027,10000038,10000023,10000007,10000031,10000042,10000017,10000047,10000016,10000036,10000004,10000025,10000035,10000021,10000013],"requested_unix":1484027821,"requested_nano":22089394},"lucky_num":10000001,"timestamp":1484027824,"date":"2017-01-10"}`))
	if err != nil {
		b.Error(err)
		b.Skip()
	}
	storeEvent := connection.MakeEvent("test.data", rd, time.Now())

	roundN := b.N
	cntChan := make(chan int, roundN)
	go func() {
		for i := 0; i < roundN; i++ {
			go func() {
				cntChan <- hub.publish(storeEvent)
			}()
		}
	}()

	for i := 0; i < roundN; i++ {
		b.Log("ok:", <-cntChan)
	}

	for _, c := range sc {
		if err := c.Err(); err != nil {
			b.Error(err)
		}
	}
}
