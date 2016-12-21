package event

import "testing"

func TestEventType(t *testing.T) {
	data := map[Event]string{
		Event("pong"):     "pong",
		Event("x.1"):      "x",
		Event("yy.a.b.c"): "yy",
		Event("*.x"):      "*",
		Event("*"):        "*",
	}

	for ev, et := range data {
		if ev.Type() != et {
			t.Errorf("type expect %s but: %s\n", et, ev.Type())
		}
	}
}

func TestEventPongMatch(t *testing.T) {
	if !PONG.Match(PONG) {
		t.Error(PONG, "MUST match", PONG)
	}
}

func TestEventMatchs(t *testing.T) {
	e := Event("x.*")
	matchTrue := []string{
		"x.",
		"x.a",
		"x...",
	}
	for _, s := range matchTrue {
		if !e.Match(Event(s)) {
			t.Error(e, s, " MUST match!")
		}
	}
	matchFalse := []string{
		"a.x.1",
		".x.a",
	}
	for _, s := range matchFalse {
		if e.Match(Event(s)) {
			t.Error(e, s, " MUST not match!")
		}
	}
	e = Event("*.x")
	matchTrue = []string{
		"a.x",
		"b.x",
		".x",
	}
	for _, s := range matchTrue {
		if !e.Match(Event(s)) {
			t.Error(e, s, " MUST match!")
		}
	}
	matchFalse = []string{
		".x.",
		".x.a",
		"x.x.b",
	}
	for _, s := range matchFalse {
		if e.Match(Event(s)) {
			t.Error(e, s, " MUST not match!")
		}
	}

}
