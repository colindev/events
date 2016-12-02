package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	l := log.New(os.Stdout, "[hub]", logFlags)
	hub, err := NewHub(env, l)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	if err := hub.ListenAndServe(quit, env.Addr); err != nil {
		log.Println(err)
	}
}
