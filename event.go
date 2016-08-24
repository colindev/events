package events

import (
	"regexp"
	"strings"
)

// Event is string of event name
// and have some method for match
type Event string

// Type fetch event first part of string
func (ev Event) Type() string {
	return strings.SplitN(string(ev), ".", 2)[0]
}

// Match test if match another event
func (ev Event) Match(event Event) bool {
	str := strings.Replace(string(ev), ".", "\\.", -1)
	str = strings.Replace(str, "*", ".*", -1)
	re := regexp.MustCompile("^" + str + "$")

	return re.MatchString(string(event))
}

const (
	// PONG is event for redis ping/pong
	PONG Event = "pong"
)
