package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {

	hub := &Hub{
		RWMutex: &sync.RWMutex{},
		m:       map[string]*Conn{},
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	err := ListenAndServe(quit, ":8000", func(c *Conn) {

		defer c.close()
		defer c.w.Flush()

		log.Printf("%#v\n", c.conn.RemoteAddr().String())

		for {
			line, err := c.readLine()
			log.Printf("<- client: %v = %s %v\n", line, line, err)
			if err != nil {
				return
			}

			/*
				$xxx = auth
				:n = len(event:{...})
				event:{...}
				+channal
			*/
			switch line[0] {
			case '$':
				if err := hub.Auth(c, line[1:]); err != nil {
					log.Println(err)
					c.w.Write([]byte("!" + err.Error()))
					return
				}
			case '+':
				c.subscribe()
			case ':':
				n, err := parseLen(line[1:])
				if err != nil {
					log.Println(err)
					c.w.Write([]byte("!" + err.Error()))
					return
				}

				log.Printf("length: %d\n", n)

				p := make([]byte, n)
				_, err = io.ReadFull(c.r, p)
				if err != nil {
					log.Println(err)
					c.w.Write([]byte("!" + err.Error()))
					return
				}

				// TODO 存起來
				eventName, eventData := parseEvent(p)
				log.Println("receive:", eventName, string(eventData))
				// 去除換行
				c.readLine()

			}
			if err := c.w.Flush(); err != nil {
				log.Println(err)
			}

		}
	})

	if err != nil {
		log.Println(err)
	}
}

func parseLen(p []byte) (int64, error) {

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

func parseEvent(p []byte) (string, []byte) {

	var index int
	ev := []byte{}

	for i, b := range p {
		if b == ':' {
			index = i + 1
			break
		}
		ev = append(ev, b)
	}

	return string(ev), p[index:]
}

type Hub struct {
	*sync.RWMutex
	m map[string]*Conn
}

func (h *Hub) Auth(c *Conn, p []byte) error {

	h.Lock()
	defer h.Unlock()

	if len(p) <= 1 {
		return errors.New("empty id")
	}

	id := string(p[1:])

	if _, exists := h.m[id]; exists {
		return fmt.Errorf("hub: duplicate auth id(%s)", id)
	}

	h.m[id] = c

	return nil
}

func (h *Hub) Quit(conn *Conn) {
	h.Lock()
	defer h.Unlock()

	for id, c := range h.m {
		if c == conn {
			delete(h.m, id)
			conn.close()
		}
	}
}

type HandlerFunc func(c *Conn)

type Conn struct {
	conn     net.Conn
	w        *bufio.Writer
	r        *bufio.Reader
	channals map[string]*regexp.Regexp
}

func (c *Conn) readLine() ([]byte, error) {
	return c.r.ReadSlice('\n')
}

func (c *Conn) subscribe(p []byte) {
	// TODO replace *
}

func (c *Conn) close() error {
	return c.conn.Close()
}

func ListenAndServe(quit <-chan os.Signal, addr string, handle HandlerFunc) error {

	network := "tcp"

	tcpAddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(network, tcpAddr)
	if err != nil {
		return err
	}

	cc := make(chan net.Conn, 10)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("conn: ", err)
				continue
			}
			cc <- conn
		}
	}()

	for {
		select {
		case s := <-quit:
			log.Printf("Receive os.Signal %s\n", s)
			return listener.Close()

		case conn := <-cc:
			go handle(&Conn{
				conn: conn,
				w:    bufio.NewWriter(conn),
				r:    bufio.NewReader(conn),
			})

		}

	}
}
