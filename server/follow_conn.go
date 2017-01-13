package main

import (
	"fmt"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/connection"
	"github.com/colindev/events/event"
	"github.com/colindev/events/store"
)

type followConn struct {
	Conn
	hub  *Hub
	addr string
}

// 遮蔽方法
func (f *followConn) SendEvent(e string) {}

// 複寫
func (f *followConn) ReadLine() (line []byte, err error) {
	// 遮蔽 join/leave
	line, err = f.Conn.ReadLine()
	if err != nil {
		f.hub.Println("conn read error:", err)
		t := time.Now()
		// run forever
		for {
			if err := Follow(f.hub, f.addr, t.Unix()); err == nil {
				break
			}
			time.Sleep(time.Second * 3)
		}
		return
	}
	if len(line) == 0 {
		return
	}

	switch line[0] {
	case connection.CEvent:
		p, e := f.ReadLen(line[1:])
		if e != nil {
			err = fmt.Errorf("error from ReadLen %v", e)
			break
		}

		eventName, compressedData, e := connection.ParseEvent(p)
		s, _ := event.Uncompress(compressedData)
		f.hub.Printf("from following %s: %s %s %v", f.Conn.RemoteAddr(), eventName, s, e)
		if e != nil {
			err = fmt.Errorf("error from ParseEvent %v", e)
			break
		}
		switch eventName {
		case event.Join:
			var auth store.Auth
			err = event.Unmarshal(s, &auth)
			if err != nil {
				err = fmt.Errorf("error from Unmarshal join %v", err)
			}
			f.hub.store.NewAuth(&auth)
		case event.Leave:
			var auth store.Auth
			err = event.Unmarshal(s, &auth)
			if err != nil {
				err = fmt.Errorf("error from Unmarshal Leave %v", err)
			}
			f.hub.store.UpdateAuth(&auth)
		case event.Connected: // ignore
		default:
			storeEvent := connection.MakeEvent(eventName, compressedData, time.Now())
			f.hub.store.Events <- storeEvent
			f.hub.publish(storeEvent)
		}
	}

	// 跳過 hub.handler 的所有處理, 但反應所有 error
	return nil, err
}

// Follow return Conn of server
func Follow(hub *Hub, addr string, since int64) error {
	cc, err := client.Dial("", addr)
	if err != nil {
		return fmt.Errorf("follow dial error: %v", err)
	}
	if err := cc.Auth(connection.Readable); err != nil {
		return fmt.Errorf("follow auth error: %v", err)
	}
	if err := cc.Subscribe("*"); err != nil {
		return fmt.Errorf("follow subscribe error: %v", err)
	}

	// client conn 連線登入完就不用了
	sc := newConn(cc.Conn(), time.Now())
	if err := hub.auth(sc, MessageAuth{Flags: connection.Writable | connection.Readable}); err != nil {
		return fmt.Errorf("follow register error: %v", err)
	}
	sc.SetAuthed(true)

	go hub.handle(&followConn{
		Conn: sc,
		hub:  hub,
		addr: addr,
	})

	cc.Recover(since, 0)
	return nil
}
