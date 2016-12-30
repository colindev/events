package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/osenv"
	"github.com/joho/godotenv"
)

var (
	version  string
	appName  string
	env      = &Env{}
	logFlags = log.LstdFlags | log.Lshortfile
)

func init() {

	showVer := flag.Bool("v", false, "version")
	flag.StringVar(&env.path, "env", ".env", "environment file")
	flag.Parse()

	if *showVer {
		fmt.Println(version)
		os.Exit(0)
	}

	env.version = version
	log.SetPrefix(fmt.Sprintf("[%s %s]", version, appName))
	log.SetFlags(logFlags)
}

func main() {
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

	l := log.New(os.Stdout, "[hub]", logFlags)
	hub, err := NewHub(env, l)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 0)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	followers := []Conn{}
	if env.Follow != "" {
		// 移花接木
		cc, err := client.Dial("", env.Follow)
		if err != nil {
			log.Fatal("follow dial error: ", err)
		}
		if err := cc.Auth(client.Readable); err != nil {
			log.Fatal("follow auth error:", err)
		}
		if err := cc.Subscribe("*"); err != nil {
			log.Fatal("follow subscribe error:", err)
		}
		// TODO 處理斷線問題

		c := newConn(cc.Conn(), time.Now())
		hub.auth(c, []byte(fmt.Sprintf(":%d", client.Writable|client.Readable)))
		followers = append(followers, c)
	}

	if err := hub.ListenAndServe(quit, env.Addr, followers...); err != nil {
		log.Println(err)
	}
}
