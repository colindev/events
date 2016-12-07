package store

import (
	"testing"
	"time"
)

func TestAuth(t *testing.T) {

	s, err := New(Config{
		AuthDSN:    "file::memory:?cache=shared",
		EventDSN:   "file::memory:?cache=shared",
		Debug:      true,
		GCDuration: "1m",
	})
	if err != nil {
		t.Error(err)
		t.Skip()
	}

	a, err := s.GetLast("game")
	if err != nil {
		t.Error(err)
		t.Skip()
	}
	t.Log(a)

	// 存入一筆新的
	a.ConnectedAt = 123
	s.NewAuth(a)

	a, err = s.GetLast("game")
	if err != nil {
		t.Error(err)
	}
	t.Log(a)
	if a.ConnectedAt != 123 {
		t.Error("GetLast error")
	}

}

func TestEvent(t *testing.T) {

	s, err := New(Config{
		AuthDSN:    "file::memory:?cache=shared",
		EventDSN:   "file::memory:?cache=shared",
		Debug:      true,
		GCDuration: "1m",
	})
	if err != nil {
		t.Error(err)
		t.Skip()
	}

	at := []int64{4, 3, 2, 1}
	g := []string{"xx", "xx", "yy", "yy"}
	for i, h := range []string{"a", "b", "c", "d"} {
		ev := &Event{
			Hash:       h,
			Prefix:     g[i],
			ReceivedAt: at[i],
		}
		s.Events <- ev
	}

	go func() {

		var expect []Event

		expect = []Event{
			0: Event{Hash: "c", ReceivedAt: 2, Prefix: "yy"},
			1: Event{Hash: "b", ReceivedAt: 3, Prefix: "xx"},
			2: Event{Hash: "a", ReceivedAt: 4, Prefix: "xx"},
		}

		i := 0
		err = s.EachEvents(func(ev *Event) {

			time.Sleep(time.Millisecond * 50)

			if expect[i].Hash != ev.Hash {
				t.Errorf("expect %d Hash=%s, but %s", i, expect[i].Hash, ev.Hash)
			}
			i++
		}, nil, 2, 0)
		if err != nil {
			t.Error("EachEvents error: ", err)
		}

		expect = []Event{
			0: Event{Hash: "d", ReceivedAt: 1, Prefix: "yy"},
			1: Event{Hash: "c", ReceivedAt: 2, Prefix: "yy"},
		}

		i = 0
		err = s.EachEvents(func(ev *Event) {

			time.Sleep(time.Millisecond * 50)

			if expect[i].Hash != ev.Hash {
				t.Errorf("expect %d Hash=%s, but %s", i, expect[i].Hash, ev.Hash)
			}
			i++
		}, []string{"yy"}, 1, 0)
		if err != nil {
			t.Error("EachEvents error: ", err)
		}

	}()

	s.Close()
}
