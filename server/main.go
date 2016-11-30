package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func writeLen(w io.Writer, prefix byte, n int) error {
	var x [32]byte
	x[len(x)-1] = '\n'
	x[len(x)-2] = '\r'
	i := len(x) - 3
	for {
		x[i] = byte('0' + n%10)
		i--
		n = n / 10
		if n == 0 {
			break
		}
	}
	x[i] = prefix
	log.Println(x)
	log.Println(x[i:])
	_, err := w.Write(x[i:])
	return err
}

func writeString(w io.Writer, s string) error {
	writeLen(w, '$', len(s))
	w.Write([]byte(s))
	_, err := w.Write([]byte("\r\n"))

	return err
}

func main() {

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	err := ListenAndServe(quit, ":8000", func(c *Conn) {

		defer c.conn.Close()

		log.Printf("%#v\n", c.conn.RemoteAddr().String())

		for {
			b, err := c.r.ReadLine()
			log.Printf("<- client: %v = %s %v\n", b, b, err)
			if err != nil {
				return
			}
		}
	})

	if err != nil {
		log.Println(err)
	}
}

/*
$ = auth
:n = len(hash:{...})
hash:{...}
*/
type Hub struct {
	*sync.RWMutex
	m map[*Conn]bool
}

func (h *Hub) Auth(line []byte) error {

	// TODO

	return nil
}

func (h *Hub) Append(c *Conn) {
	h.Lock()
	defer h.Unlock()

	h.m[c] = true
}

func (h *Hub) Delete(c *Conn) {
	h.Lock()
	defer h.Unlock()

	delete(h.m, c)
}

type HandlerFunc func(c *Conn)

type Conn struct {
	conn    net.Conn
	w       *bufio.Writer
	r       *bufio.Reader
	session string
}

func (c *Conn) auth() error {

	line, err := c.readLine()

}

func (c *Conn) readLine() ([]byte, error) {
	return c.r.ReadSlice('\n')
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
