package main

import "github.com/colindev/events/event"

// Message contain all data
type Message struct {
	Action byte
	Value  interface{}
	Error  error
}

// MessageAuth contain auth request data
type MessageAuth struct {
	Name  string
	Flags int
}

// MessageRecover contain recover request data
type MessageRecover struct {
	Since int64
	Until int64
}

// MessageSubscribe contain subscribe request data
type MessageSubscribe struct {
	Channel string
}

// MessageUnsubscribe contain unsubscribe request data
type MessageUnsubscribe struct {
	Channel string
}

// MessagePing contain ping request data
type MessagePing struct {
	Payload []byte
}

// MessageInfo contain info request data
type MessageInfo struct{}

// MessageEvent contain event request data
type MessageEvent struct {
	To      string
	Name    event.Event
	RawData event.RawData
}
