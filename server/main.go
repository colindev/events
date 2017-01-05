package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	if env.Follow != "" {
		Follow(hub, env.Follow, 0)
	}

	if err := hub.ListenAndServe(quit, env.Addr); err != nil {
		log.Println(err)
	}
}
