package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/colindev/events/client"
	"github.com/colindev/events/event"
	"github.com/colindev/events/listener"
)

func main() {

	eventName := event.Event("test.event")

	l := listener.New(func() (client.Conn, error) {
		return client.Dial("test-listener", ":6300")
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGQUIT)

	go func() {
		for {
			now := time.Now()
			rand.Seed(int64(now.Nanosecond()))
			n := rand.Intn(3)
			log.Println("rand: ", n)
			if n == 1 {
				log.Println("\033[31m================= test crash ================\033[m")
				panic(fmt.Sprintf("test panic %s", now))
			}
			du := time.Minute*time.Duration(rand.Intn(3)) + time.Nanosecond*time.Duration(rand.Intn(999999))
			log.Println("next rand: ", du)
			time.Sleep(du)
		}
	}()

	l.On(event.Ready, func(ev event.Event, v event.RawData) {
		l.Recover(0, 0)
	}).On(event.Connected, func(ev event.Event, v event.RawData) {
		log.Println(ev)
	}).On(event.Disconnected, func(ev event.Event, v event.RawData) {
		log.Println(ev, v.String())
	}).On(eventName, func(ev event.Event, v event.RawData) {
		fmt.Println(ev, v.String())
	}).RunForever(quit, time.Second*5, eventName).WaitHandler()
}
