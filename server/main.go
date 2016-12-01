package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/colindev/osenv"
	"github.com/joho/godotenv"
)

var (
	env      = &Env{}
	logFlags = log.LstdFlags | log.Lshortfile
)

func init() {

	flag.StringVar(&env.path, "env", ".env", "environment file")
	flag.Parse()

	log.SetFlags(logFlags)
	if err := godotenv.Overload(env.path); err != nil {
		log.Fatal(err)
	}
	if err := osenv.LoadTo(env); err != nil {
		log.Fatal(err)
	}
	log.Println(env.String())
	if !env.Debug {
		logFlags = log.LstdFlags
		log.SetFlags(logFlags)
	}
}

func main() {

	var err error

	l := log.New(os.Stdout, "[hub]", logFlags)
	hub, err := NewHub(env, l)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	err = listenAndServe(quit, env.Addr, hub.Handler)

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

func listenAndServe(quit <-chan os.Signal, addr string, handle HandlerFunc) error {

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
			go handle(newConn(conn, time.Now()))

		}

	}
}
