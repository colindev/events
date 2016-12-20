package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type Line struct {
	raw    string
	cnt    int
	prefix string
	next   *Line
}

func (l *Line) String() string {

	ret := fmt.Sprintf("%s %s", l.prefix, l.raw)
	switch l.cnt {
	case 0:
		return fmt.Sprintf("\033[31m%s (%d)\033[m", ret, l.cnt)
	case 1:
		return fmt.Sprintf("\033[32m%s (%d)\033[m", ret, l.cnt)
	}
	return fmt.Sprintf("\033[35m%s (%d)\033[m", ret, l.cnt)
}

func main() {

	var (
		launcherFile string
		listenerFile string
	)

	flag.StringVar(&launcherFile, "launcher", "./launcher.log", "launcher log file")
	flag.StringVar(&listenerFile, "listener", "./listener.log", "listener log file")
	flag.Parse()

	fLauncher, err := os.Open(launcherFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fLauncher.Close()

	fListener, err := os.Open(listenerFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fListener.Close()

	cache := map[string]*Line{}

	var startLine, endLine, currentLine *Line
	bufLauncher := bufio.NewReader(fLauncher)
	cntLauncher := 0
	for {
		line, _, err := bufLauncher.ReadLine()
		if err != nil {
			break
		}

		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}

		cntLauncher++
		L := &Line{
			raw: s,
		}
		if startLine == nil {
			startLine = L
		}
		endLine = L
		if currentLine != nil {
			currentLine.next = L
		}
		currentLine = L

		cache[L.raw] = L
	}

	notInFire := []string{}
	bufListener := bufio.NewReader(fListener)
	cntListener := 0
	cntCrashed := 0
	cntRecover := 0
	cntConnected := 0
	cntDisconnected := 0
	var startListen, endListen string
	statusPrefix := " "
	for {
		line, _, err := bufListener.ReadLine()
		if err != nil {
			break
		}
		if len(line) > 0 && line[0] == '=' {
			switch strings.TrimSpace(string(line[1:])) {
			case " ":
				break
			case "crash":
				statusPrefix = "x"
				cntCrashed++
			case "ready":
				statusPrefix = "o"
				cntRecover++
			case "connected":
				statusPrefix = "|"
				cntConnected++
			case "disconnected":
				statusPrefix = "-"
				cntDisconnected++
			default:
				statusPrefix = "?"
			}
			continue
		}

		cntListener++

		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}

		if startListen == "" {
			startListen = s
		}
		endListen = s

		if l, exists := cache[s]; !exists {
			notInFire = append(notInFire, s)
		} else {
			l.cnt++
			l.prefix = statusPrefix
		}
	}

	cntMiss := 0
	cntDup := 0
	L := startLine
	prevPrefix := ""
	for {
		if L == nil {
			break
		}
		if L.prefix == "" {
			L.prefix = prevPrefix
		} else {
			prevPrefix = L.prefix
		}
		fmt.Println(L.String())
		if L.cnt == 0 {
			cntMiss++
		} else if L.cnt > 1 {
			cntDup++
		}
		L = L.next
	}

	fmt.Printf("miss: %d duplicate: %d crashed: %d recover: %d connected: %d disconnected: %d\n",
		cntMiss, cntDup, cntCrashed, cntRecover, cntConnected, cntDisconnected)
	fmt.Printf("[fire]: %d \033[32mstart:\033[m %s \033[32mend:\033[m %s\n", cntLauncher, startLine, endLine)
	fmt.Printf("[rece]: %d \033[32mstart:\033[m %s \033[32mend:\033[m %s\n", cntListener, startListen, endListen)

	fmt.Println("not in fire:")
	for _, s := range notInFire {
		fmt.Println(s)
	}
}
