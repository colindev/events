package main

import (
	"bytes"
	"strconv"

	"github.com/colindev/events/connection"
)

func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func isBetween(a, b, c int64) bool {
	n, m := min(b, c), max(b, c)

	return a >= n && a <= m
}

func makeLen(prefix byte, n int) *bytes.Buffer {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(prefix)
	buf.WriteString(strconv.Itoa(n))
	buf.Write(connection.EOL)
	return buf
}

func makeReply(m string) *bytes.Buffer {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(connection.CReply)
	buf.WriteString(m)
	return buf
}

func makeError(err error) *bytes.Buffer {
	buf := makeLen(connection.CErr, len(err.Error()))
	buf.WriteString(err.Error())
	return buf
}

func makePong(ping []byte) *bytes.Buffer {
	buf := makeLen(connection.CPong, len(ping))
	buf.Write(ping)
	return buf
}

func makeEvent(e string) *bytes.Buffer {
	buf := makeLen(connection.CEvent, len(e))
	buf.WriteString(e)
	return buf
}
