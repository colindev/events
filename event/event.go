package event

import (
	"regexp"
	"strings"
)

const (
	// PONG is event for redis ping/pong
	PONG Event = "pong"
)

type (
	// Event is string of event name
	// and have some method for match
	Event string
)

// Type fetch event first part of string
func (ev Event) Type() string {
	return strings.SplitN(ev.String(), ".", 2)[0]
}

// Match test if match another event
func (ev Event) Match(event Event) bool {
	str := strings.Replace(ev.String(), ".", "\\.", -1)
	str = strings.Replace(str, "*", ".*", -1)
	re := regexp.MustCompile("^" + str + "$")

	return re.MatchString(event.String())
}

func (ev Event) String() string {
	return string(ev)
}

// Bytes convert event to []byte
func (ev Event) Bytes() []byte {
	return []byte(ev)
}
