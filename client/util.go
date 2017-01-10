package client

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/colindev/events/event"
	"github.com/colindev/events/store"
)

// ParseLen of socket stream
func ParseLen(p []byte) (int64, error) {

	raw := string(p)

	if len(p) == 0 {
		return -1, errors.New("length error:" + raw)
	}

	var negate bool
	if p[0] == '-' {
		negate = true
		p = p[1:]
		if len(p) == 0 {
			return -1, errors.New("length error:" + raw)
		}
	}

	var n int64
	for _, b := range p {

		if b == '\r' || b == '\n' {
			break
		}
		n *= 10
		if b < '0' || b > '9' {
			return -1, errors.New("not number:" + string(b))
		}

		n += int64(b - '0')
	}

	if negate {
		n = -n
	}

	return n, nil
}

// ParseTargetAndLen from socket stream
func ParseTargetAndLen(p []byte) (appName string, length int64, err error) {
	i := bytes.IndexByte(p, ':')
	if i == -1 {
		err = errors.New("schema error: expect {app name}:{len}")
		return
	}

	appName = string(p[:i])
	length, err = ParseLen(p[i+1:])
	return
}

// ParseEvent from socket stream
func ParseEvent(p []byte) (event.Event, event.RawData, error) {

	var (
		sp  int
		err error
	)
	box := [30]byte{}

	for i, b := range p {
		if b == ':' {
			sp = i
			break
		}
		box[i] = b
	}
	ev := box[:sp]
	data := p[sp+1:]

	if len(ev) > 30 {
		err = errors.New("event name over 30 char")
	} else if len(ev) == 0 {
		err = errors.New("event name is empty")
	} else if len(data) == 0 {
		err = errors.New("event data empty")
	}

	return event.Event(ev), event.RawData(data), err
}

// ParseSinceUntil from socket stream
func ParseSinceUntil(p []byte) (since int64, until int64) {
	s := strings.SplitN(string(p), ":", 2)
	since, _ = strconv.ParseInt(s[0], 10, 64)
	if len(s) == 2 {
		until, _ = strconv.ParseInt(s[1], 10, 64)
	}
	return
}

// MakeEventStream build stream from event data
func MakeEventStream(ev event.Event, rd event.RawData) []byte {
	buf := bytes.NewBuffer(ev.Bytes())
	buf.WriteByte(':')
	buf.Write(rd.Bytes())

	return buf.Bytes()
}

// MakeEvent build store.Event
func MakeEvent(ev event.Event, rd event.RawData, t time.Time) *store.Event {
	p := MakeEventStream(ev, rd)
	return &store.Event{
		Hash:       fmt.Sprintf("%x", sha1.Sum(p)),
		Name:       ev.String(),
		Prefix:     ev.Type(),
		Length:     len(p),
		Raw:        string(p),
		ReceivedAt: t.Unix(),
	}
}

// WriteLen to buffer writer
func WriteLen(w *bufio.Writer, prefix byte, n int) error {
	w.WriteByte(prefix)
	w.WriteString(strconv.Itoa(n))
	_, err := w.Write(EOL)
	return err
}

// WriteTargetAndLen to buffer writer
func WriteTargetAndLen(w *bufio.Writer, prefix byte, target string, n int) error {
	w.WriteByte(prefix)
	w.WriteString(target)
	w.WriteByte(':')
	w.WriteString(strconv.Itoa(n))
	_, err := w.Write(EOL)
	return err
}

// WriteEvent to socket
func WriteEvent(w *bufio.Writer, p []byte) error {
	WriteLen(w, CEvent, len(p))
	_, err := w.Write(p)
	return err
}

// WriteEventTo tosocket for specific target
func WriteEventTo(w *bufio.Writer, name string, p []byte) error {
	WriteTargetAndLen(w, CTarget, name, len(p))
	_, err := w.Write(p)
	return err
}

// WriteInfo request to socket
func WriteInfo(w *bufio.Writer) error {
	return w.WriteByte(CInfo)
}

// WritePing to socket
func WritePing(w *bufio.Writer, m string) error {
	WriteLen(w, CPing, len(m))
	_, err := w.WriteString(m)
	return err
}

// WriteAuth to socket
func WriteAuth(w *bufio.Writer, name string, flags int) error {
	w.WriteByte(CAuth)
	_, err := w.WriteString(fmt.Sprintf("%s:%d", name, flags))
	return err
}

// WriteRecover to socket
func WriteRecover(w *bufio.Writer, since, until int64) error {
	w.WriteByte(CRecover)
	_, err := w.WriteString(strconv.FormatInt(since, 10) + ":" + strconv.FormatInt(until, 10))
	return err
}

// WriteSubscribe to socket
func WriteSubscribe(w *bufio.Writer, chans ...string) (err error) {

	for _, ch := range chans {
		w.WriteByte(CAddChan)
		w.WriteString(ch)
		_, err = w.Write(EOL)
	}

	return
}

// WriteUnsubscribe to socket
func WriteUnsubscribe(w *bufio.Writer, chans ...string) (err error) {

	for _, ch := range chans {
		w.WriteByte(CDelChan)
		w.WriteString(ch)
		_, err = w.Write(EOL)
	}

	return
}
