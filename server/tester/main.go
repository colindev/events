package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

func main() {

	addr := flag.String("addr", ":6300", "server address")
	c := flag.Int("c", 500, "num of client")

	flag.Parse()

	wg := &sync.WaitGroup{}

	for i := 0; i < *c; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", *addr)
			if err != nil {
				log.Println(err)
				return
			}
			rand.Seed(int64(time.Now().Nanosecond()))
			n := time.Duration(rand.Intn(10)) * time.Second
			time.Sleep(n)
			log.Println(i, n, conn.Close())

		}(i)
	}

	wg.Wait()
}
